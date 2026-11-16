// Package platform selects built-in implementations at compile time. Backends
// are Go interfaces, not dynamically loaded plugins.
package platform

import (
	"afishk/internal/audio"
	"afishk/internal/desktop"
)

type Backends struct {
	Windows desktop.WindowSource
	Capture desktop.Capturer
	Input   desktop.Input
	Audio   audio.Capture
	close   func()
}

// Close releases the desktop session. Stop the audio manager before calling it.
func (b *Backends) Close() {
	if b.close != nil {
		b.close()
	}
}
