// Package desktop guards manual actions independently of the desktop backend.
package desktop

import (
	"context"
	"image"
)

// Window identifies a single native window; ID is opaque to the controller.
type Window struct {
	Executable string `json:"executable,omitempty"`
	ID         string `json:"id"`
	Class      string `json:"class"`
	Title      string `json:"title"`
}

// WindowSource must fail when focus or unlocked-session state cannot be verified.
// It must never activate a window to satisfy a request.
type WindowSource interface {
	Active(context.Context) (Window, error)
}

// Capturer captures only the supplied native window ID, never a desktop fallback.
type Capturer interface {
	Capture(context.Context, string) (image.Image, error)
}

// Input sends one F press/release to the verified target. Implementations must
// never retry key-down; a best-effort key-up is permitted after failure.
type Input interface {
	SendF(context.Context, Window) error
}
