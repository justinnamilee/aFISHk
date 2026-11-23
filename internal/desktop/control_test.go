package desktop

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type fakeDesktop struct {
	w                     Window
	activeErr, captureErr error
}

func (f *fakeDesktop) Active(context.Context) (Window, error) { return f.w, f.activeErr }
func (f *fakeDesktop) Capture(context.Context, string) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, 2, 2)), f.captureErr
}
func controller(t *testing.T) (*Controller, *fakeDesktop) {
	t.Helper()
	f := &fakeDesktop{w: Window{ID: "id", Class: "wow.exe", Title: "World of Warcraft"}}
	return &Controller{Windows: f, Capture: f, Class: "wow.exe", Title: "World of Warcraft", CacheDir: t.TempDir()}, f
}
func TestFishRejectsWrongOrUnknownFocus(t *testing.T) {
	for _, kind := range []string{"class", "title", "missing", "error"} {
		t.Run(kind, func(t *testing.T) {
			c, f := controller(t)
			switch kind {
			case "class":
				f.w.Class = "terminal"
			case "title":
				f.w.Title = "Other"
			case "missing":
				f.w.ID = ""
			case "error":
				f.activeErr = errors.New("no compositor")
			}
			calls := 0
			c.Input = inputFunc(func(context.Context, Window) error { calls++; return nil })
			if c.Fish(context.Background()) == nil || calls != 0 {
				t.Fatal("unverified target must never receive input")
			}
		})
	}
}
func TestFishOneTapAndThrottle(t *testing.T) {
	c, _ := controller(t)
	var calls []Window
	c.Input = inputFunc(func(_ context.Context, w Window) error { calls = append(calls, w); return nil })
	if err := c.Fish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].ID != "id" {
		t.Fatal(calls)
	}
	if c.Fish(context.Background()) == nil || len(calls) != 1 {
		t.Fatal("rapid tap was queued or sent")
	}
}
func TestBusyInputIsRejected(t *testing.T) {
	c, _ := controller(t)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Fish(context.Background()) == nil {
		t.Fatal("expected busy")
	}
}
func TestScreenshotPreservesLatestOnFailure(t *testing.T) {
	c, f := controller(t)
	if err := c.Screenshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(c.CacheDir, "latest.png")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f.captureErr = errors.New("permission denied")
	if c.Screenshot(context.Background()) == nil {
		t.Fatal("expected capture failure")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("lost previous capture")
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && stat.Mode().Perm() != 0600 {
		t.Fatal(stat.Mode())
	}
	files, err := os.ReadDir(c.CacheDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("unwanted history: %v %v", files, err)
	}
}

type inputFunc func(context.Context, Window) error

func (f inputFunc) SendF(ctx context.Context, w Window) error { return f(ctx, w) }

func TestFailedInputIsNotRetried(t *testing.T) {
	c, _ := controller(t)
	calls := 0
	c.Input = inputFunc(func(ctx context.Context, _ Window) error {
		calls++
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 2*time.Second {
			t.Error("missing input deadline")
		}
		return errors.New("failed")
	})
	if c.Fish(context.Background()) == nil || calls != 1 {
		t.Fatal("expected single failed call")
	}
}

func TestMissingBackendsFailClosed(t *testing.T) {
	c, _ := controller(t)
	if c.Fish(context.Background()) == nil {
		t.Fatal("missing input accepted")
	}
	c.Capture = nil
	if c.Screenshot(context.Background()) == nil {
		t.Fatal("missing capture accepted")
	}
	c.Windows = nil
	if _, err := c.Target(context.Background()); err == nil {
		t.Fatal("missing window source accepted")
	}
}

func TestCaptureReceivesVerifiedID(t *testing.T) {
	c, f := controller(t)
	calls := 0
	c.Capture = captureFunc(func(_ context.Context, id string) (image.Image, error) {
		calls++
		if id != f.w.ID {
			t.Fatal("wrong capture target")
		}
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
	})
	if err := c.Screenshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.activeErr = errors.New("locked or unknown")
	if c.Screenshot(context.Background()) == nil || calls != 1 {
		t.Fatal("unverified capture")
	}
}

type captureFunc func(context.Context, string) (image.Image, error)

func (f captureFunc) Capture(ctx context.Context, id string) (image.Image, error) { return f(ctx, id) }

func TestFishRejectsWrongExecutable(t *testing.T) {
	c, f := controller(t)
	c.Executable = "C:\\Games\\WoW.exe"
	f.w.Executable = "C:\\Other\\WoW.exe"
	calls := 0
	c.Input = inputFunc(func(context.Context, Window) error { calls++; return nil })
	if c.Fish(context.Background()) == nil || calls != 0 {
		t.Fatal("wrong executable received input")
	}
	f.w.Executable = "c:\\games\\wow.exe"
	if err := c.Fish(context.Background()); err != nil || calls != 1 {
		t.Fatal("same Windows path rejected", err)
	}
}
