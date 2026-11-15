package desktop

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Backend interface {
	Active(context.Context) (Window, error)
	Capture(context.Context, string) (image.Image, error)
}
type Controller struct {
	Backend                         Backend
	Class, Title, Ydotool, CacheDir string
	mu                              sync.Mutex
	lastInput                       time.Time
	Run                             func(context.Context, string, ...string) error
}

func (c *Controller) Target(ctx context.Context) (Window, error) {
	w, err := c.Backend.Active(ctx)
	if err != nil {
		return w, err
	}
	if w.Class != c.Class || w.Title != c.Title || w.ID == "" {
		return Window{}, errors.New("WoW is not focused")
	}
	return w, nil
}
func run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
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
	if _, err := c.Target(ctx); err != nil {
		return err
	}
	fn := c.Run
	if fn == nil {
		fn = run
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	c.lastInput = time.Now()
	// Linux KEY_F = 33. A single invocation sends down and up, no repetition.
	if err := fn(ctx, c.Ydotool, "key", "--key-delay", "20", "33:1", "33:0"); err != nil {
		// Best effort key-up if the client died after sending key-down. Never retry down.
		release, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		_ = fn(release, c.Ydotool, "key", "33:0")
		return fmt.Errorf("ydotool failed (key-up attempted): %w", err)
	}
	return nil
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
	img, err := c.Backend.Capture(ctx, w.ID)
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
