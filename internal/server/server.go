// Package server exposes a fixed set of same-origin companion actions.
package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"afishk/internal/config"
	"github.com/pion/webrtc/v4"
)

//go:embed web/*
var web embed.FS

type Desktop interface {
	Fish(context.Context) error
	Screenshot(context.Context) error
}
type Audio interface {
	Start(context.Context, webrtc.SessionDescription) (webrtc.SessionDescription, error)
	Stop()
	KeepAlive() bool
}
type Server struct {
	Config  config.Config
	Desktop Desktop
	Audio   Audio
	token   string
	csrf    string
}

func New(c config.Config, d Desktop, a Audio) (*Server, error) {
	s := &Server{Config: c, Desktop: d, Audio: a}
	if c.TokenFile != "" {
		b, err := os.ReadFile(c.TokenFile)
		if err != nil {
			return nil, err
		}
		s.token = strings.TrimSpace(string(b))
		if len(s.token) < 24 {
			return nil, errors.New("token_file must contain at least 24 characters")
		}
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	s.csrf = hex.EncodeToString(nonce)
	return s, nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(web, "web")
	mux.Handle("GET /", http.FileServer(http.FS(static)))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"connected": true, "csrf": s.csrf, "audio": "Desktop audio → WebRTC; start to verify availability"})
	})
	mux.HandleFunc("POST /api/fish", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Desktop.Fish(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, map[string]string{"message": "Fishing key sent"})
	})
	mux.HandleFunc("POST /api/screenshot", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := s.Desktop.Screenshot(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, map[string]string{"message": "Fresh screenshot saved"})
	})
	mux.HandleFunc("GET /api/screenshot.png", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(filepath.Join(s.Config.CacheDir, "latest.png"))
		if err != nil {
			http.Error(w, "No successful capture yet", http.StatusNotFound)
			return
		}
		defer func() { _ = f.Close() }()
		info, err := f.Stat()
		if err != nil {
			http.Error(w, "Cannot read capture", http.StatusInternalServerError)
			return
		}
		r.Header.Del("If-Modified-Since")
		r.Header.Del("If-None-Match")
		w.Header().Set("Content-Type", "image/png")
		http.ServeContent(w, r, "latest.png", info.ModTime(), f)
	})
	mux.HandleFunc("POST /api/audio/offer", func(w http.ResponseWriter, r *http.Request) {
		var offer webrtc.SessionDescription
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&offer); err != nil {
			http.Error(w, "Invalid offer", http.StatusBadRequest)
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			http.Error(w, "Expected one offer", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		answer, err := s.Audio.Start(ctx, offer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, answer)
	})
	mux.HandleFunc("POST /api/audio/stop", func(w http.ResponseWriter, _ *http.Request) {
		s.Audio.Stop()
		writeJSON(w, map[string]bool{"stopped": true})
	})
	mux.HandleFunc("POST /api/audio/keepalive", func(w http.ResponseWriter, _ *http.Request) {
		if !s.Audio.KeepAlive() {
			http.Error(w, "Audio disconnected", http.StatusConflict)
			return
		}
		writeJSON(w, map[string]bool{"connected": true})
	})
	return s.protect(mux)
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func equal(a, b string) bool {
	x, y := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}
func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' blob:; media-src 'self' blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		origin := s.Config.PublicOrigin
		if origin == "" {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			origin = scheme + "://" + s.Config.Listen
		}
		if strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://") != r.Host {
			http.Error(w, "Unexpected host", http.StatusForbidden)
			return
		}
		if s.token != "" {
			_, pass, ok := r.BasicAuth()
			if !ok || !equal(pass, s.token) {
				w.Header().Set("WWW-Authenticate", `Basic realm="aFISHk"`)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if got := r.Header.Get("Origin"); got != "" && got != origin {
				http.Error(w, "Wrong origin", http.StatusForbidden)
				return
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" || !equal(r.Header.Get("X-CSRF-Token"), s.csrf) {
				http.Error(w, "Invalid request token; reconnect", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
