package setup

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"afishk/internal/config"
	"afishk/internal/desktop"
)

//go:embed web/*
var wizardFiles embed.FS

type webWizard struct {
	path, origin, csrf string
	windows            desktop.WindowSource
	defaults           config.Config
	mu                 sync.Mutex
	target             desktop.Window
	saved, detecting   bool
	done               chan struct{}
	once               sync.Once
}

type webSettings struct {
	Listen  string `json:"listen"`
	Origin  string `json:"origin"`
	AudioIP string `json:"audio_ip"`
	Pulse   string `json:"pulse_source"`
	TLS     string `json:"tls"`
	Cert    string `json:"cert"`
	Key     string `json:"key"`
}

// RunWeb is a temporary, loopback-only first-run wizard. It never replaces an
// existing configuration, starts desktop control or opens a LAN setup listener.
func RunWeb(ctx context.Context, path string, source desktop.WindowSource, open func(string) error, out io.Writer) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(path); !os.IsNotExist(err) {
		return errors.New("configuration already exists or cannot be accessed")
	}
	defaults, err := config.Defaults()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer func() { _ = ln.Close() }()
	nonce := make([]byte, 32)
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	w := &webWizard{path: path, origin: "http://" + ln.Addr().String(), csrf: hex.EncodeToString(nonce), windows: source, defaults: defaults, done: make(chan struct{})}
	requests, cancelRequests := context.WithCancel(ctx)
	defer cancelRequests()
	srv := &http.Server{Handler: w.handler(), BaseContext: func(net.Listener) context.Context { return requests }, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	result := make(chan error, 1)
	go func() { result <- srv.Serve(ln) }()
	// The URL contains no password or capability; the page gets a per-run CSRF
	// nonce via same-origin JSON. Remote websites cannot read it or submit changes.
	_, _ = fmt.Fprintln(out, "Open local setup:", w.origin)
	_ = open(w.origin)
	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()
	select {
	case <-w.done:
	case <-ctx.Done():
		err = ctx.Err()
	case <-timeout.C:
		err = errors.New("local setup timed out; launch setup again")
	case serveErr := <-result:
		return serveErr
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cancelRequests()
	if shutdownErr := srv.Shutdown(shutdown); shutdownErr != nil {
		_ = srv.Close()
	}
	return err
}

func (w *webWizard) handler() http.Handler {
	mux := http.NewServeMux()
	assets, _ := fs.Sub(wizardFiles, "web")
	mux.Handle("GET /", http.FileServer(http.FS(assets)))
	mux.HandleFunc("GET /api/init", func(rw http.ResponseWriter, r *http.Request) {
		addresses := []string{}
		if list, err := net.InterfaceAddrs(); err == nil {
			for _, addr := range list {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() {
					addresses = append(addresses, ipnet.IP.String())
				}
			}
		}
		respond(rw, map[string]any{"csrf": w.csrf, "platform": runtime.GOOS, "addresses": addresses, "pulse_source": w.defaults.PulseSource})
	})
	mux.HandleFunc("POST /api/detect", func(rw http.ResponseWriter, r *http.Request) {
		w.mu.Lock()
		if w.saved || w.detecting {
			w.mu.Unlock()
			http.Error(rw, "Setup is busy or already saved", http.StatusConflict)
			return
		}
		w.detecting = true
		w.target = desktop.Window{}
		w.mu.Unlock()
		defer func() { w.mu.Lock(); w.detecting = false; w.mu.Unlock() }()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-r.Context().Done():
			return
		}
		if w.windows == nil {
			http.Error(rw, "Desktop inspection unavailable", http.StatusConflict)
			return
		}
		target, err := w.windows.Active(r.Context())
		if err != nil {
			http.Error(rw, err.Error(), http.StatusConflict)
			return
		}
		if target.ID == "" || target.Class == "" || target.Title == "" {
			http.Error(rw, "Cannot verify the game window", http.StatusConflict)
			return
		}
		w.mu.Lock()
		w.target = target
		w.mu.Unlock()
		respond(rw, target)
	})
	mux.HandleFunc("POST /api/save", func(rw http.ResponseWriter, r *http.Request) {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.saved || w.detecting || w.target.ID == "" {
			http.Error(rw, "Identify the focused game before saving; an existing config cannot be replaced", http.StatusConflict)
			return
		}
		var values webSettings
		dec := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 16<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&values); err != nil {
			http.Error(rw, "Invalid settings", http.StatusBadRequest)
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			http.Error(rw, "Expected one settings object", http.StatusBadRequest)
			return
		}
		c, err := w.configuration(values)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := Save(c, w.path)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusConflict)
			return
		}
		w.saved = true
		respond(rw, map[string]string{"origin": c.PublicOrigin, "token": token, "config": w.path})
	})
	mux.HandleFunc("POST /api/finish", func(rw http.ResponseWriter, r *http.Request) {
		w.mu.Lock()
		saved := w.saved
		w.mu.Unlock()
		if !saved {
			http.Error(rw, "Save configuration first", http.StatusConflict)
			return
		}
		respond(rw, map[string]bool{"saved": true})
		w.once.Do(func() { close(w.done) })
	})
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Cache-Control", "no-store")
		rw.Header().Set("Referrer-Policy", "no-referrer")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		rw.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'none'; base-uri 'none'")
		rw.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		rw.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		if r.Host != strings.TrimPrefix(w.origin, "http://") || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(rw, "Unexpected host or site", http.StatusForbidden)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if r.Header.Get("Origin") != w.origin || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(w.csrf)) != 1 {
				http.Error(rw, "Invalid setup request", http.StatusForbidden)
				return
			}
		}
		mux.ServeHTTP(rw, r)
	})
}

func (w *webWizard) configuration(v webSettings) (config.Config, error) {
	c := w.defaults
	c.Listen, c.PublicOrigin, c.ICEAddress = v.Listen, v.Origin, v.AudioIP
	c.GameClass, c.GameTitle, c.GameExecutable = w.target.Class, w.target.Title, w.target.Executable
	if runtime.GOOS == "linux" {
		c.PulseSource = v.Pulse
	}
	c.TokenFile = filepath.Join(filepath.Dir(w.path), "token")
	scheme := "http"
	switch v.TLS {
	case "no":
	case "proxy":
		scheme = "https"
	case "builtin":
		scheme = "https"
		if !filepath.IsAbs(v.Cert) || !filepath.IsAbs(v.Key) {
			return c, errors.New("TLS file paths must be absolute")
		}
		if _, err := tls.LoadX509KeyPair(v.Cert, v.Key); err != nil {
			return c, err
		}
		c.TLSCert, c.TLSKey = v.Cert, v.Key
	default:
		return c, errors.New("select a TLS mode")
	}
	u, err := url.Parse(c.PublicOrigin)
	if err != nil || u.Scheme != scheme || u.Hostname() == "" {
		return c, errors.New("browser address must match the selected TLS mode")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && ip.IsUnspecified() {
		return c, errors.New("browser address cannot be a wildcard")
	}
	if ip := net.ParseIP(c.ICEAddress); ip.To4() == nil || ip.IsUnspecified() {
		return c, errors.New("enter the desktop IPv4 reachable by your phone")
	}
	return c, c.Validate()
}
func respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
