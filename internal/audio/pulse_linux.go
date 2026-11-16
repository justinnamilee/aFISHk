package audio

import (
	"context"
	"io"
	"os/exec"
)

// PulseCapture bridges a Pulse/PipeWire monitor into Opus RTP using FFmpeg.
type PulseCapture struct {
	FFmpeg string
	Source string
}

func (p PulseCapture) Start(ctx context.Context, target string) (Process, error) {
	// Keep 20 ms packet pacing, in-band FEC and immediate muxer flushing.
	cmd := exec.CommandContext(ctx, p.FFmpeg, "-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", "pulse", "-i", p.Source, "-vn", "-c:a", "libopus", "-application", "audio",
		"-fec:a", "1", "-packet_loss", "10", "-frame_duration", "20", "-ar", "48000", "-ac", "2", "-b:a", "96000",
		"-payload_type", "111", "-flush_packets", "1", "-f", "rtp", target)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
