package addon

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAndRejectUnsafeArchive(t *testing.T) {
	for _, entry := range []string{"FishFry/FishFry.toc", "../FishFry.toc", "/FishFry.toc", "Other/Other.toc", "FishFry/../escape.toc"} {
		t.Run(entry, func(t *testing.T) {
			dir := t.TempDir()
			wow := filepath.Join(dir, "game")
			if err := os.MkdirAll(filepath.Join(wow, "Interface", "AddOns"), 0755); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "addon.zip")
			f, err := os.Create(p)
			if err != nil {
				t.Fatal(err)
			}
			z := zip.NewWriter(f)
			w, err := z.Create(entry)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write([]byte("## Interface: 16001")); err != nil {
				t.Fatal(err)
			}
			if err = z.Close(); err != nil {
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
			err = Install(p, wow)
			if entry == "FishFry/FishFry.toc" {
				if err != nil {
					t.Fatal(err)
				}
				if Install(p, wow) == nil {
					t.Fatal("overwrote existing addon")
				}
			} else if err == nil {
				t.Fatal("accepted unsafe archive")
			}
		})
	}
}
