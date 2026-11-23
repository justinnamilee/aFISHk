package desktop

import (
	"afishk/internal/keybind"
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestYdotoolPressAndFailureRelease(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var calls [][]string
		y := &Ydotool{Path: "ydotool", run: func(ctx context.Context, path string, args ...string) error {
			if path != "ydotool" || ctx.Err() != nil {
				t.Fatal("invalid command context")
			}
			calls = append(calls, args)
			if fail {
				return errors.New("offline")
			}
			return nil
		}}
		keys, err := keybind.Parse("Ctrl+Shift+G")
		if err != nil {
			t.Fatal(err)
		}
		err = y.SendKeys(context.Background(), Window{ID: "id"}, keys)
		if (err != nil) != fail {
			t.Fatal(err)
		}
		want := [][]string{{"key", "--key-delay", "20", "29:1", "42:1", "34:1", "34:0", "42:0", "29:0"}}
		if fail {
			want = append(want, []string{"key", "34:0", "42:0", "29:0"})
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("got %v, want %v", calls, want)
		}
	}
}
