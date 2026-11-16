package desktop

import (
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
		err := y.SendF(context.Background(), Window{ID: "id"})
		if (err != nil) != fail {
			t.Fatal(err)
		}
		want := [][]string{{"key", "--key-delay", "20", "33:1", "33:0"}}
		if fail {
			want = append(want, []string{"key", "33:0"})
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("got %v, want %v", calls, want)
		}
	}
}
