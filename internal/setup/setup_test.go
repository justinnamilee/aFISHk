package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"afishk/internal/config"
)

func TestSetup(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "afishk", "config.json")
	answers := "10.6.6.34:8787\nno\nhttp://suzuki:8787\n\nsteam_app_0\n\n\n"
	var out bytes.Buffer
	if err := Run(strings.NewReader(answers), &out, path); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicOrigin != "http://suzuki:8787" || c.ICEAddress != "10.6.6.34" || c.AllowUnauthenticatedLAN {
		t.Fatalf("bad config: %+v", c)
	}
	for _, p := range []string{path, c.TokenFile} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("%s mode %v", p, info.Mode())
		}
	}
	token, err := os.ReadFile(c.TokenFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(token))) != 64 {
		t.Fatal("expected 256-bit hex token")
	}
	if strings.Contains(out.String(), strings.TrimSpace(string(token))) {
		t.Fatal("token leaked to terminal")
	}
	if err := Run(strings.NewReader(answers), &out, path); err == nil {
		t.Fatal("overwrote config")
	}
	after, _ := os.ReadFile(c.TokenFile)
	if !bytes.Equal(token, after) {
		t.Fatal("rotated token on rerun")
	}
	// A different config in the same directory reuses the existing private token.
	if err := Run(strings.NewReader(answers), &out, filepath.Join(filepath.Dir(path), "other.json")); err != nil {
		t.Fatal(err)
	}
	after, _ = os.ReadFile(c.TokenFile)
	if !bytes.Equal(token, after) {
		t.Fatal("rotated existing token")
	}
}

func TestSetupRejectsIncompleteOrInvalidInput(t *testing.T) {
	for _, answers := range []string{
		"", "10.6.6.34:8787\n", "hostname:8787\n", "0.0.0.0:8787\nno\nhttp://suzuki:8787\n\nsteam_app_0\n\n\n",
		"10.6.6.34:8787\nproxy\nhttp://suzuki:8787\n",
		"10.6.6.34:8787\nno\nhttp://suzuki:8787\n\n\n\n\n",
	} {
		t.Run(answers, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := Run(strings.NewReader(answers), &bytes.Buffer{}, path); err == nil {
				t.Fatal("accepted invalid/incomplete answers")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("left config after cancellation")
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), "token")); !os.IsNotExist(err) {
				t.Fatal("left token after cancellation")
			}
		})
	}
}

func TestSetupProxy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	answers := "127.0.0.1:8787\nproxy\nhttps://fish.example.test\n10.6.6.34\nsteam_app_0\n\n\n"
	if err := Run(strings.NewReader(answers), &bytes.Buffer{}, path); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.TLSCert != "" || c.TLSKey != "" || c.PublicOrigin != "https://fish.example.test" {
		t.Fatalf("bad proxy configuration: %+v", c)
	}
}

func TestSetupRejectsPublicToken(t *testing.T) {
	dir := t.TempDir()
	token := filepath.Join(dir, "token")
	if err := os.WriteFile(token, []byte(strings.Repeat("a", 64)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(token, 0644); err != nil {
		t.Fatal(err)
	}
	answers := "\nno\n\n\nsteam_app_0\n\n\n"
	if err := Run(strings.NewReader(answers), &bytes.Buffer{}, filepath.Join(dir, "config.json")); err == nil {
		t.Fatal("accepted public token")
	}
}
