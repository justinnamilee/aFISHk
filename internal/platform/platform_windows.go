package platform

import (
	"afishk/internal/audio"
	"afishk/internal/config"
	"afishk/internal/desktop"
)

func Open(c config.Config) (*Backends, error) {
	n, err := desktop.OpenWindows()
	if err != nil {
		return nil, err
	}
	return &Backends{Windows: n, Capture: n, Input: n, Audio: audio.WASAPICapture{FFmpeg: c.FFmpeg}}, nil
}
