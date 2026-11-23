//go:build linux

package audio

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"afishk/internal/config"

	"github.com/pion/transport/v5"
	"github.com/pion/transport/v5/stdnet"
	"github.com/pion/webrtc/v4"
)

// Exercise real SDP, ICE, DTLS/SRTP and FFmpeg Opus encoding without a desktop.
// Only the Pulse input is replaced with a synthetic sine source.
func TestWebRTCAudioAndCleanup(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg is required for the audio integration test")
	}
	wrapper := filepath.Join(t.TempDir(), "ffmpeg-test")
	// Preserve the real encoder/muxer arguments so unsupported options fail this
	// test too. Replace only the desktop input and pace the synthetic source.
	script := fmt.Sprintf(`#!/bin/bash
set -eu
args=("$@")
for ((i=0; i<${#args[@]}; i++)); do
  if [[ ${args[i]} == -f && ${args[i+1]:-} == pulse ]]; then args[i+1]=lavfi; fi
  if [[ ${args[i]} == -i ]]; then args[i+1]=sine=frequency=440:sample_rate=48000; fi
done
exec %q -re "${args[@]}"
`, ffmpeg)
	if err = os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	m := New(config.Config{UDPMin: 49200, UDPMax: 49250}, PulseCapture{FFmpeg: wrapper, Source: "unused"})
	defer m.Close()
	m.configure = isolatedNetwork
	var settings webrtc.SettingEngine
	isolatedNetwork(&settings)
	pc, err := webrtc.NewAPI(webrtc.WithSettingEngine(settings)).NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pc.Close() }()
	if _, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		t.Fatal(err)
	}
	received := make(chan error, 1)
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Verify a second of continuous 20 ms audio, not just one startup packet.
		var sequence uint16
		var timestamp uint32
		for i := 0; i < 50; i++ {
			packet, _, e := track.ReadRTP()
			if e != nil {
				received <- e
				return
			}
			if len(packet.Payload) == 0 || (i > 0 && (packet.SequenceNumber != sequence+1 || packet.Timestamp-timestamp != 960)) {
				received <- fmt.Errorf("discontinuous or empty audio at packet %d", i)
				return
			}
			sequence, timestamp = packet.SequenceNumber, packet.Timestamp
		}
		received <- nil
	})
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	done := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("client ICE timeout")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	answer, err := m.Start(ctx, *pc.LocalDescription())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Start(ctx, *pc.LocalDescription()); err == nil {
		t.Fatal("accepted simultaneous audio peer")
	}
	if err = pc.SetRemoteDescription(answer); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-received:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no audio received")
	}
	if !m.KeepAlive() {
		t.Fatal("live audio lease missing")
	}
	m.Stop()
	if m.KeepAlive() {
		t.Fatal("stopped audio still live")
	}
	m.mu.Lock()
	remaining := m.session
	m.mu.Unlock()
	if remaining != nil {
		t.Fatal("session leaked")
	}
}
func TestInvalidOfferAndShutdown(t *testing.T) {
	m := New(config.Config{}, nil)
	defer m.Close()
	if _, err := m.Start(context.Background(), webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer}); err == nil {
		t.Fatal("accepted answer")
	}
	m.Close()
	if _, err := m.Start(context.Background(), webrtc.SessionDescription{Type: webrtc.SDPTypeOffer}); err == nil || !strings.Contains(err.Error(), "shutting down") {
		t.Fatal(err)
	}
}

// Avoid interface enumeration privileges in restricted build environments while
// retaining real loopback UDP sockets, FFmpeg and the full WebRTC media stack.
type loopbackNet struct{ *stdnet.Net }

func (n loopbackNet) Interfaces() ([]*transport.Interface, error) {
	iface := transport.NewInterface(net.Interface{Index: 1, Name: "lo", Flags: net.FlagUp | net.FlagLoopback})
	iface.AddAddress(&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.CIDRMask(8, 32)})
	return []*transport.Interface{iface}, nil
}
func isolatedNetwork(se *webrtc.SettingEngine) {
	se.SetNet(loopbackNet{Net: &stdnet.Net{}})
	se.SetIncludeLoopbackCandidate(true)
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
}

// This backend has no FFmpeg dependency: verify Manager owns cancellation,
// failed-start cleanup, lease expiry and producer-exit cleanup for any backend.
type fakeCapture struct {
	fail        bool
	ctx         context.Context
	exit        chan struct{}
	waited      chan struct{}
	cleanupGate chan struct{}
}

func (f *fakeCapture) Start(ctx context.Context, _ string) (Process, error) {
	f.ctx = ctx
	if f.fail {
		return nil, fmt.Errorf("capture unavailable")
	}
	return f, nil
}
func (f *fakeCapture) Wait() error {
	defer close(f.waited)
	select {
	case <-f.ctx.Done():
	case <-f.exit:
	}
	if f.cleanupGate != nil {
		<-f.cleanupGate
	}
	return nil
}

func TestShutdownHoldsSessionUntilProducerExits(t *testing.T) {
	f := &fakeCapture{exit: make(chan struct{}), waited: make(chan struct{}), cleanupGate: make(chan struct{})}
	m := New(config.Config{UDPMin: 49200, UDPMax: 49250}, f)
	m.configure = isolatedNetwork
	defer m.Close()
	var release sync.Once
	defer release.Do(func() { close(f.cleanupGate) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	offer := testOffer(t)
	if _, err := m.Start(ctx, offer); err != nil {
		t.Fatal(err)
	}
	stopped := make(chan struct{})
	go func() { m.Stop(); close(stopped) }()
	select {
	case <-f.ctx.Done():
	case <-ctx.Done():
		t.Fatal("shutdown did not cancel the producer")
	}
	select {
	case <-stopped:
		t.Fatal("Stop returned before cleanup")
	default:
	}
	if _, err := m.Start(ctx, offer); err == nil {
		t.Fatal("accepted a second producer before cleanup finished")
	}
	if m.KeepAlive() {
		t.Fatal("renewed a closing session")
	}
	release.Do(func() { close(f.cleanupGate) })
	select {
	case <-stopped:
	case <-ctx.Done():
		t.Fatal("Stop did not finish after producer cleanup")
	}
}

func testOffer(t *testing.T) webrtc.SessionDescription {
	t.Helper()
	var se webrtc.SettingEngine
	isolatedNetwork(&se)
	pc, err := webrtc.NewAPI(webrtc.WithSettingEngine(se)).NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	if _, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		t.Fatal(err)
	}
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	done := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ICE timeout")
	}
	return *pc.LocalDescription()
}

func TestCaptureLifecycle(t *testing.T) {
	for _, mode := range []string{"start-failure", "stop", "close", "exit", "lease"} {
		t.Run(mode, func(t *testing.T) {
			f := &fakeCapture{fail: mode == "start-failure", exit: make(chan struct{}), waited: make(chan struct{})}
			m := New(config.Config{UDPMin: 49200, UDPMax: 49250}, f)
			m.configure = isolatedNetwork
			defer m.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := m.Start(ctx, testOffer(t))
			if mode == "start-failure" {
				if err == nil || m.KeepAlive() || f.ctx.Err() == nil {
					t.Fatal("failed capture leaked session/context")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "stop":
				m.Stop()
			case "close":
				m.Close()
			case "exit":
				close(f.exit)
			case "lease":
				m.mu.Lock()
				m.session.timer.Reset(time.Millisecond)
				m.mu.Unlock()
			}
			if mode == "stop" || mode == "close" {
				select {
				case <-f.waited:
				default:
					t.Fatal("shutdown returned before producer cleanup")
				}
			}
			select {
			case <-f.ctx.Done():
			case <-ctx.Done():
				t.Fatal("producer not cancelled")
			}
			select {
			case <-f.waited:
			case <-ctx.Done():
				t.Fatal("producer not reaped")
			}
			if m.KeepAlive() {
				t.Fatal("session lease survived cleanup")
			}
		})
	}
}
