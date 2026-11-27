package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"afishk/internal/config"
	"afishk/internal/desktop"
	"rsc.io/qr"
)

func testWizard(t *testing.T) *webWizard {
	t.Helper()
	defaults, err := config.Defaults()
	if err != nil {
		t.Fatal(err)
	}
	return &webWizard{path: filepath.Join(t.TempDir(), "config.json"), origin: "http://127.0.0.1:12345", csrf: strings.Repeat("x", 64), defaults: defaults, target: desktop.Window{ID: "verified", Class: "wow.exe", Title: "World of Warcraft", Executable: "C:\\Games\\WoW.exe"}, done: make(chan struct{})}
}
func wizardRequest(w *webWizard, method, path, body string) *http.Request {
	r := httptest.NewRequest(method, w.origin+path, strings.NewReader(body))
	if method == http.MethodPost {
		r.Header.Set("Origin", w.origin)
		r.Header.Set("X-CSRF-Token", w.csrf)
	}
	return r
}
func TestWizardRejectsCrossOriginAndRebinding(t *testing.T) {
	for _, kind := range []string{"host", "origin", "csrf", "cross-site"} {
		t.Run(kind, func(t *testing.T) {
			w := testWizard(t)
			r := wizardRequest(w, "POST", "/api/save", "{}")
			switch kind {
			case "host":
				r.Host = "evil.example:12345"
			case "origin":
				r.Header.Set("Origin", "https://evil.example")
			case "csrf":
				r.Header.Del("X-CSRF-Token")
			case "cross-site":
				r.Header.Set("Sec-Fetch-Site", "cross-site")
			}
			out := httptest.NewRecorder()
			w.handler().ServeHTTP(out, r)
			if out.Code != 403 {
				t.Fatalf("%d %s", out.Code, out.Body.String())
			}
			if _, err := os.Stat(w.path); !os.IsNotExist(err) {
				t.Fatal("unexpected config write")
			}
		})
	}
}
func TestWizardSavesVerifiedTargetAndNeverOverwrites(t *testing.T) {
	w := testWizard(t)
	body := `{"fish_key":"Ctrl+Shift+4","lure_key":"Ctrl+8","listen":"192.168.1.10:8787","origin":"http://192.168.1.10:8787","audio_ip":"192.168.1.10","pulse_source":"@DEFAULT_MONITOR@","tls":"no"}`
	out := httptest.NewRecorder()
	w.handler().ServeHTTP(out, wizardRequest(w, "POST", "/api/save", body))
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	var saved struct {
		Token string `json:"token"`
		QR    bool   `json:"qr"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Token) != 64 {
		t.Fatal("missing random password")
	}
	out = httptest.NewRecorder()
	w.handler().ServeHTTP(out, wizardRequest(w, "GET", "/api/qr", ""))
	expected, err := qr.Encode("http://192.168.1.10:8787", qr.M)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.QR || out.Code != http.StatusOK || out.Header().Get("Content-Type") != "image/png" || !bytes.Equal(out.Body.Bytes(), expected.PNG()) {
		t.Fatal("expected a PNG QR code encoding only the control page URL")
	}
	if _, err := png.Decode(out.Body); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(w.path)
	if err != nil {
		t.Fatal(err)
	}
	if c.LureKey != "Ctrl+8" {
		t.Fatal("lure binding was not saved")
	}
	if c.FishKey != "Ctrl+Shift+4" {
		t.Fatal("fishing binding was not saved")
	}
	if c.GameClass != w.target.Class || c.GameTitle != w.target.Title || c.GameExecutable != w.target.Executable {
		t.Fatal("unverified target persisted")
	}
	original, err := os.ReadFile(w.path)
	if err != nil {
		t.Fatal(err)
	}
	out = httptest.NewRecorder()
	w.handler().ServeHTTP(out, wizardRequest(w, "POST", "/api/save", body))
	if out.Code != 409 {
		t.Fatal("allowed overwrite")
	}
	after, err := os.ReadFile(w.path)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("config changed")
	}
	out = httptest.NewRecorder()
	w.handler().ServeHTTP(out, wizardRequest(w, "POST", "/api/finish", "{}"))
	select {
	case <-w.done:
	default:
		t.Fatal("wizard did not finish")
	}
}
func TestWizardRequiresDetectionAndValidNetworkSettings(t *testing.T) {
	w := testWizard(t)
	w.target = desktop.Window{}
	out := httptest.NewRecorder()
	w.handler().ServeHTTP(out, wizardRequest(w, "POST", "/api/save", "{}"))
	if out.Code != 409 {
		t.Fatal("accepted unknown target")
	}
	w = testWizard(t)
	for _, v := range []webSettings{
		{Listen: "192.168.1.1:0", Origin: "http://192.168.1.1", AudioIP: "192.168.1.1", TLS: "no"},
		{Listen: "192.168.1.1:8787", Origin: "http://0.0.0.0:8787", AudioIP: "192.168.1.1", TLS: "no"},
		{Listen: "192.168.1.1:8787", Origin: "http://192.168.1.1:8787", AudioIP: "0.0.0.0", TLS: "no"},
		{Listen: "192.168.1.1:8787", Origin: "http://192.168.1.1:8787", AudioIP: "192.168.1.1", TLS: "proxy"},
	} {
		if _, err := w.configuration(v); err == nil {
			t.Fatal("accepted invalid configuration")
		}
	}
}
func TestWizardExistingConfigurationDoesNotOpenListener(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	opened := false
	err := RunWeb(context.Background(), path, nil, func(string) error { opened = true; return nil }, &bytes.Buffer{}, false)
	if err == nil || opened {
		t.Fatal("existing configuration opened setup")
	}
}

func TestWizardOverwritePreservesTokenAndAdvancedSettings(t *testing.T) {
	w := testWizard(t)
	initial := w.defaults
	initial.GameClass, initial.GameTitle, initial.GameExecutable = w.target.Class, w.target.Title, w.target.Executable
	initial.UDPMin, initial.UDPMax = 50200, 50201
	initial.TokenFile = filepath.Join(filepath.Dir(w.path), "custom-token")
	token, err := Save(initial, w.path, false)
	if err != nil {
		t.Fatal(err)
	}
	w.overwrite = true
	w.defaults, err = setupConfig(w.path, w.overwrite)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"listen":"192.168.1.10:8787","origin":"https://fish.example.test","audio_ip":"192.168.1.10","pulse_source":"@DEFAULT_MONITOR@","tls":"proxy"}`
	out := httptest.NewRecorder()
	w.handler().ServeHTTP(out, wizardRequest(w, "POST", "/api/save", body))
	if out.Code != http.StatusOK {
		t.Fatal(out.Code, out.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(out.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(w.path)
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicOrigin != "https://fish.example.test" || c.TokenFile != initial.TokenFile || c.UDPMin != initial.UDPMin || c.UDPMax != initial.UDPMax || saved["token"] != token {
		t.Fatal("overwrite lost advanced settings or changed the password")
	}
}
