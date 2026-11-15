package server

import (
	"afishk/internal/audio"
	"afishk/internal/config"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type fakeDesktop struct{ calls int }

func (f *fakeDesktop) Fish(context.Context) error       { f.calls++; return nil }
func (f *fakeDesktop) Screenshot(context.Context) error { return nil }
func TestProtectAllRoutesAndCSRF(t *testing.T) {
	dir := t.TempDir()
	token := "test-password-long-enough-for-authentication"
	p := filepath.Join(dir, "token")
	if err := os.WriteFile(p, []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	c := config.Config{Listen: "127.0.0.1:8787", TokenFile: p, CacheDir: dir}
	d := &fakeDesktop{}
	a := audio.New(c)
	defer a.Close()
	s, err := New(c, d, a)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	for _, path := range []string{"/", "/health", "/app.js", "/api/status", "/api/screenshot.png", "/api/fish", "/api/audio/offer", "/api/audio/stop"} {
		method := "GET"
		if path == "/api/fish" || path == "/api/audio/offer" || path == "/api/audio/stop" {
			method = "POST"
		}
		r := httptest.NewRequest(method, "http://127.0.0.1:8787"+path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("%s unauthenticated: %d", path, w.Code)
		}
	}
	for _, tc := range []struct {
		name, csrf, origin, host string
		want                     int
	}{
		{"missing-csrf", "", "http://127.0.0.1:8787", "127.0.0.1:8787", 403},
		{"wrong-origin", s.csrf, "https://evil.example", "127.0.0.1:8787", 403},
		{"dns-rebinding", s.csrf, "http://evil.example", "evil.example", 403},
		{"valid", s.csrf, "http://127.0.0.1:8787", "127.0.0.1:8787", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "http://127.0.0.1:8787/api/fish", nil)
			r.Host = tc.host
			r.SetBasicAuth("fish", token)
			r.Header.Set("X-CSRF-Token", tc.csrf)
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing cache control")
			}
		})
	}
	if d.calls != 1 {
		t.Fatal("rejected requests reached desktop", d.calls)
	}
}
func TestScreenshotNoCacheAndNoTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "latest.png"), []byte("png-test"), 0600); err != nil {
		t.Fatal(err)
	}
	c := config.Config{Listen: "127.0.0.1:8787", CacheDir: dir}
	a := audio.New(c)
	defer a.Close()
	s, err := New(c, &fakeDesktop{}, a)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:8787/api/screenshot.png", nil)
	r.Header.Set("If-Modified-Since", "Wed, 01 Jan 2099 00:00:00 GMT")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Body.String() != "png-test" {
		t.Fatalf("stale image response: %d %q", w.Code, w.Body.String())
	}
}
