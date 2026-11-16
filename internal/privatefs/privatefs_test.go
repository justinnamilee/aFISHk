package privatefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteNewPreservesExistingSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "afishk", "token")
	if err := WriteNew(path, []byte("original secret")); err != nil {
		t.Fatal(err)
	}
	if err := CheckFile(path); err != nil {
		t.Fatal(err)
	}
	if err := WriteNew(path, []byte("replacement")); !os.IsExist(err) {
		t.Fatalf("expected existing-file rejection: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original secret" {
		t.Fatalf("secret changed: %v", err)
	}
	if err := CheckFile(filepath.Dir(path)); err == nil {
		t.Fatal("directory accepted as a token file")
	}
}
