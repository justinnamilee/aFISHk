package audio

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	script := fmt.Sprintf("#!/bin/bash\nset -eu\nfor last; do :; done\nexec %q -nostdin -hide_banner -loglevel error -re -f lavfi -i sine=frequency=440:sample_rate=48000 -c:a libopus -ac 2 -ar 48000 -frame_duration 20 -payload_type 111 -f rtp \"$last\"\n", ffmpeg)
	if err = os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	m := New(config.Config{FFmpeg: wrapper, PulseSource: "unused", UDPMin: 49200, UDPMax: 49250})
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
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) { _, _, e := track.ReadRTP(); received <- e })
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
	m := New(config.Config{})
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
