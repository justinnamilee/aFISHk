package desktop

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Controller struct {
	Windows                WindowSource
	Capture                Capturer
	Input                  Input
	Class, Title, CacheDir string
	mu                     sync.Mutex
	lastInput              time.Time
}

func (c *Controller) Target(ctx context.Context) (Window, error) {
	if c.Windows == nil {
		return Window{}, errors.New("window backend unavailable")
	}
	w, err := c.Windows.Active(ctx)
	if err != nil {
		return w, err
	}
	if w.Class != c.Class || w.Title != c.Title || w.ID == "" {
		return Window{}, errors.New("WoW is not focused")
	}
	return w, nil
}

func (c *Controller) Fish(ctx context.Context) error {
	// Drop busy requests instead of queueing delayed keypresses.
	if !c.mu.TryLock() {
		return errors.New("desktop action busy; tap again when ready")
	}
	defer c.mu.Unlock()
	if time.Since(c.lastInput) < 350*time.Millisecond {
		return errors.New("tap too fast")
	}
	w, err := c.Target(ctx)
	if err != nil {
		return err
	}
	if c.Input == nil {
		return errors.New("input backend unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	c.lastInput = time.Now()
	return c.Input.SendF(ctx, w)
}
func (c *Controller) Screenshot(ctx context.Context) error {
	if !c.mu.TryLock() {
		return errors.New("desktop action busy")
	}
	defer c.mu.Unlock()
	w, err := c.Target(ctx)
	if err != nil {
		return err
	}
	if c.Capture == nil {
		return errors.New("capture backend unavailable")
	}
	img, err := c.Capture.Capture(ctx, w.ID)
	if err != nil {
		return err
	}
	return savePNG(c.CacheDir, img)
}
func savePNG(dir string, img image.Image) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".capture-*.png")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if err = png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, "latest.png"))
}
