package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadValidation(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		good       bool
	}{
		{"loopback", `{"game_class":"wow.exe"}`, true},
		{"vpn-without-auth", `{"listen":"100.64.1.2:8787","game_class":"wow.exe"}`, false},
		{"vpn-with-auth", `{"listen":"100.64.1.2:8787","game_class":"wow.exe","token_file":"/tmp/token"}`, true},
		{"missing-identity", `{"listen":"127.0.0.1:8787"}`, false}, {"typo", `{"game_class":"wow","litsen":"127.0.0.1:5"}`, false},
		{"partial-tls", `{"game_class":"wow","tls_cert":"/tmp/cert"}`, false},
		{"extra-object", `{"game_class":"wow"} {}`, false},
		{"bad-origin", `{"game_class":"wow","public_origin":"https://example.com/path"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				tc.body = strings.Replace(tc.body, "{", `{"game_executable":"C:\\Games\\WoW.exe",`, 1)
			}
			p := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(p, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(p)
			if (err == nil) != tc.good {
				t.Fatalf("valid=%v, error=%v", tc.good, err)
			}
		})
	}
}
