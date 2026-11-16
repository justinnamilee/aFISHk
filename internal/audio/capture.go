package audio

import "context"

// Capture starts a desktop audio producer sending Opus RTP (48 kHz, stereo,
// payload type 111) to the supplied loopback rtp:// destination. Cancellation
// must stop the producer and unblock Wait. Start failure must release resources.
// WebRTC, UDP bounds, peer limits and leases belong to Manager, not the backend.
type Capture interface {
	Start(context.Context, string) (Process, error)
}

// Process is owned by Manager, which calls Wait exactly once after Start.
type Process interface {
	Wait() error
}
