package desktop

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Ydotool sends Linux KEY_F through the separately managed input daemon.
type Ydotool struct {
	Path string
	run  func(context.Context, string, ...string) error
}

func (y *Ydotool) SendF(ctx context.Context, _ Window) error {
	fn := y.run
	if fn == nil {
		fn = func(ctx context.Context, name string, args ...string) error {
			return exec.CommandContext(ctx, name, args...).Run()
		}
	}
	if err := fn(ctx, y.Path, "key", "--key-delay", "20", "33:1", "33:0"); err != nil {
		release, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		_ = fn(release, y.Path, "key", "33:0")
		return fmt.Errorf("ydotool failed (key-up attempted): %w", err)
	}
	return nil
}
