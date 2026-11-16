// Package audio manages captured Opus RTP into one WebRTC peer.
package audio

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"afishk/internal/config"
	"github.com/pion/webrtc/v4"
)

type Manager struct {
	capture Capture
	// configure permits isolated transports in integration tests; production uses host networking.
	configure func(*webrtc.SettingEngine)
	mu        sync.Mutex
	session   *session
	config    config.Config
	closed    bool
}
type session struct {
	pc           *webrtc.PeerConnection
	cancel       context.CancelFunc
	conn         *net.UDPConn
	timer        *time.Timer
	producerDone chan struct{}
	closing      bool // guarded by Manager.mu
	once         sync.Once
}

func New(c config.Config, capture Capture) *Manager { return &Manager{config: c, capture: capture} }
func (s *session) close() {
	s.once.Do(func() {
		s.cancel()
		_ = s.conn.Close()
		_ = s.pc.Close()
		if s.producerDone != nil {
			<-s.producerDone
		}
	})
}

func (m *Manager) Start(ctx context.Context, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if offer.Type != webrtc.SDPTypeOffer {
		return webrtc.SessionDescription{}, errors.New("expected WebRTC offer")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return webrtc.SessionDescription{}, errors.New("audio is shutting down")
	}
	if m.session != nil {
		return webrtc.SessionDescription{}, errors.New("audio already in use; stop it or wait 20 seconds for lease expiry")
	}
	if m.capture == nil {
		return webrtc.SessionDescription{}, errors.New("audio capture backend unavailable")
	}
	var se webrtc.SettingEngine
	if err := se.SetEphemeralUDPPortRange(m.config.UDPMin, m.config.UDPMax); err != nil {
		return webrtc.SessionDescription{}, err
	}
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	if m.config.ICEAddress != "" {
		if err := se.SetICEAddressRewriteRules(webrtc.ICEAddressRewriteRule{
			External: []string{m.config.ICEAddress}, AsCandidateType: webrtc.ICECandidateTypeHost,
			Mode: webrtc.ICEAddressRewriteReplace,
		}); err != nil {
			return webrtc.SessionDescription{}, err
		}
	}
	if m.configure != nil {
		m.configure(&se)
	}
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		_ = pc.Close()
		return webrtc.SessionDescription{}, err
	}
	mediaCtx, cancel := context.WithCancel(context.Background())
	s := &session{pc: pc, cancel: cancel, conn: conn}
	success := false
	defer func() {
		if !success {
			s.close()
		}
	}()
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2}, "desktop", "afishk")
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	sender, err := pc.AddTrack(track)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, e := sender.Read(buf); e != nil {
				return
			}
		}
	}()
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateClosed {
			go m.remove(s)
		}
	})
	if err = pc.SetRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	gathered := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	select {
	case <-gathered:
	case <-ctx.Done():
		return webrtc.SessionDescription{}, ctx.Err()
	}
	target := fmt.Sprintf("rtp://127.0.0.1:%d?pkt_size=1200", conn.LocalAddr().(*net.UDPAddr).Port)
	process, err := m.capture.Start(mediaCtx, target)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("start audio capture: %w", err)
	}
	m.session = s
	s.timer = time.AfterFunc(20*time.Second, func() { m.remove(s) })
	s.producerDone = make(chan struct{})
	go func() { _ = process.Wait(); close(s.producerDone); m.remove(s) }()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, _, e := conn.ReadFromUDP(buf)
			if e != nil {
				return
			}
			if _, e = track.Write(buf[:n]); e != nil {
				m.remove(s)
				return
			}
		}
	}()
	success = true
	return *pc.LocalDescription(), nil
}
func (m *Manager) remove(s *session) {
	m.mu.Lock()
	s.closing = true
	if s.timer != nil {
		s.timer.Stop()
	}
	m.mu.Unlock()
	s.close()
	m.mu.Lock()
	if m.session == s {
		m.session = nil
	}
	m.mu.Unlock()
}
func (m *Manager) KeepAlive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil || m.session.closing {
		return false
	}
	m.session.timer.Reset(20 * time.Second)
	return true
}
func (m *Manager) Stop() {
	m.mu.Lock()
	s := m.session
	m.mu.Unlock()
	if s != nil {
		m.remove(s)
	}
}
func (m *Manager) Close() { m.mu.Lock(); m.closed = true; m.mu.Unlock(); m.Stop() }
