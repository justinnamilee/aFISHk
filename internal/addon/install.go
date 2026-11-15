// Package addon installs an operator-downloaded FishFry archive. Downloads and
// third-party licenses remain with the addon publisher; nothing is bundled.
package addon

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Install(archive, wowDir string) error {
	root, err := filepath.Abs(wowDir)
	if err != nil {
		return err
	}
	// Require a real WoW installation so a mistyped destination cannot create one.
	addons := filepath.Join(root, "Interface", "AddOns")
	if info, e := os.Stat(addons); e != nil || !info.IsDir() {
		return errors.New("wow-dir must contain Interface/AddOns (use the game-client directory)")
	}
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = z.Close() }()
	if len(z.File) > 10000 {
		return errors.New("too many addon archive entries")
	}
	var total uint64
	foundTOC := false
	for _, f := range z.File {
		name := strings.TrimSuffix(f.Name, "/")
		if name == "" || strings.Contains(name, "\\") || path.IsAbs(name) || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe addon archive entry: %q", f.Name)
		}
		if name != "FishFry" && !strings.HasPrefix(name, "FishFry/") {
			return fmt.Errorf("expected FishFry/ at archive root, got %q", f.Name)
		}
		if !f.FileInfo().IsDir() && !f.Mode().IsRegular() {
			return errors.New("archive contains special files")
		}
		if f.UncompressedSize64 > 64<<20 {
			return errors.New("addon file too large")
		}
		total += f.UncompressedSize64
		if total > 128<<20 {
			return errors.New("addon archive too large")
		}
		if path.Dir(name) == "FishFry" && strings.HasSuffix(strings.ToLower(name), ".toc") {
			foundTOC = true
		}
	}
	if !foundTOC {
		return errors.New("no FishFry .toc manifest found")
	}
	dest := filepath.Join(addons, "FishFry")
	if _, err = os.Lstat(dest); !os.IsNotExist(err) {
		return errors.New("FishFry destination exists; move it aside before updating")
	}
	staging, err := os.MkdirTemp(addons, ".afishk-addon-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	for _, f := range z.File {
		target := filepath.Join(staging, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err = extract(f, target); err != nil {
			return err
		}
	}
	return os.Rename(filepath.Join(staging, "FishFry"), dest)
}
func extract(f *zip.File, target string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, io.LimitReader(src, 64<<20+1))
	closeErr := dst.Close()
	if err != nil {
		return err
	}
	return closeErr
}
