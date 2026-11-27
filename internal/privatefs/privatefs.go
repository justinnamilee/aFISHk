// Package privatefs creates user-private application data on each platform.
package privatefs

import (
	"os"
	"path/filepath"
)

func Dir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	return protect(path)
}

// WriteNew never replaces an existing file. Secure its directory before writing.
func WriteNew(path string, data []byte) error {
	if err := Dir(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = protect(path)
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}

// Replace writes a private file next to the destination before replacing it.
func Replace(path string, data []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return os.ErrPermission
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".afishk-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	err = protect(f.Name())
	if err == nil {
		_, err = f.Write(data)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
