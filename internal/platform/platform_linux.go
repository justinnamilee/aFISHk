package platform

import (
	"afishk/internal/audio"
	"afishk/internal/config"
	"afishk/internal/desktop"
)

func Open(c config.Config) (*Backends, error) {
	k, err := desktop.OpenKWin()
	if err != nil {
		return nil, err
	}
	return &Backends{
		Windows: k, Capture: k, Input: &desktop.Ydotool{Path: c.Ydotool},
		Audio: audio.PulseCapture{FFmpeg: c.FFmpeg, Source: c.PulseSource}, close: k.Close,
	}, nil
}
