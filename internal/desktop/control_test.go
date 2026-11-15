package desktop

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"
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
	f := &fakeDesktop{w: Window{"id", "wow.exe", "World of Warcraft"}}
	return &Controller{Backend: f, Class: "wow.exe", Title: "World of Warcraft", Ydotool: "ydotool", CacheDir: t.TempDir()}, f
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
			c.Run = func(context.Context, string, ...string) error { calls++; return nil }
			if c.Fish(context.Background()) == nil || calls != 0 {
				t.Fatal("unverified target must never receive input")
			}
		})
	}
}
func TestFishOneTapAndThrottle(t *testing.T) {
	c, _ := controller(t)
	var calls [][]string
	c.Run = func(_ context.Context, _ string, args ...string) error { calls = append(calls, args); return nil }
	if err := c.Fish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], []string{"key", "--key-delay", "20", "33:1", "33:0"}) {
		t.Fatal(calls)
	}
	if c.Fish(context.Background()) == nil || len(calls) != 1 {
		t.Fatal("rapid tap was queued or sent")
	}
}
func TestFailedInputOnlyAttemptsRelease(t *testing.T) {
	c, _ := controller(t)
	var calls [][]string
	c.Run = func(_ context.Context, _ string, args ...string) error {
		calls = append(calls, args)
		return errors.New("offline")
	}
	if c.Fish(context.Background()) == nil {
		t.Fatal("expected error")
	}
	if len(calls) != 2 || !reflect.DeepEqual(calls[1], []string{"key", "33:0"}) {
		t.Fatal(calls)
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
	if stat.Mode().Perm() != 0600 {
		t.Fatal(stat.Mode())
	}
	files, err := os.ReadDir(c.CacheDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("unwanted history: %v %v", files, err)
	}
}
func TestSnapshotRejectsSpoofedAndStaleReplies(t *testing.T) {
	ch := make(chan Window, 1)
	k := &KWin{pending: map[string]pending{"nonce": {owner: ":1.42", result: ch}}}
	if k.Snapshot(":1.99", "nonce", "id", "wow.exe", "World of Warcraft") == nil {
		t.Fatal("accepted wrong D-Bus sender")
	}
	if k.Snapshot(":1.42", "stale", "id", "wow.exe", "World of Warcraft") == nil {
		t.Fatal("accepted stale reply")
	}
	if err := k.Snapshot(":1.42", "nonce", "id", "wow.exe", "World of Warcraft"); err != nil {
		t.Fatal(err)
	}
	select {
	case w := <-ch:
		if w.ID != "id" {
			t.Fatal(w)
		}
	default:
		t.Fatal("missing KWin reply")
	}
}
