package desktop

import (
	"afishk/internal/keybind"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Ydotool sends configured key chords through the separately managed input daemon.
type Ydotool struct {
	Path string
	run  func(context.Context, string, ...string) error
}

func (y *Ydotool) SendKeys(ctx context.Context, _ Window, keys []keybind.Key) error {
	fn := y.run
	if fn == nil {
		fn = func(ctx context.Context, name string, args ...string) error {
			return exec.CommandContext(ctx, name, args...).Run()
		}
	}
	args := []string{"key", "--key-delay", "20"}
	var releases []string
	for _, key := range keys {
		args = append(args, fmt.Sprintf("%d:1", key.Linux))
	}
	for i := len(keys) - 1; i >= 0; i-- {
		releases = append(releases, fmt.Sprintf("%d:0", keys[i].Linux))
	}
	if err := fn(ctx, y.Path, append(args, releases...)...); err != nil {
		release, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		_ = fn(release, y.Path, append([]string{"key"}, releases...)...)
		return fmt.Errorf("ydotool failed (key-up attempted): %w", err)
	}
	return nil
}
