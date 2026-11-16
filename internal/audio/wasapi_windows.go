package audio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ole32 = windows.NewLazySystemDLL("ole32.dll")
var clsidEnumerator = windows.GUID{Data1: 0xbcde0395, Data2: 0xe52f, Data3: 0x467c, Data4: [8]byte{0x8e, 0x3d, 0xc4, 0x57, 0x92, 0x91, 0x69, 0x2e}}
var iidEnumerator = windows.GUID{Data1: 0xa95664d2, Data2: 0x9614, Data3: 0x4f35, Data4: [8]byte{0xa7, 0x46, 0xde, 0x8d, 0xb6, 0x36, 0x17, 0xe6}}
var iidAudioClient = windows.GUID{Data1: 0x1cb9ad4c, Data2: 0xdbfa, Data3: 0x4c32, Data4: [8]byte{0xb1, 0x78, 0xc2, 0xf5, 0x68, 0xa7, 0x03, 0xb2}}
var iidCapture = windows.GUID{Data1: 0xc8adbd64, Data2: 0xe71e, Data3: 0x48a0, Data4: [8]byte{0xa4, 0xde, 0x18, 0x5c, 0x39, 0x5c, 0xd3, 0x17}}

type comAudio struct{ vtable *[16]uintptr }

//go:uintptrescapes
func (c *comAudio) invoke(slot int, args ...uintptr) error {
	values := append([]uintptr{uintptr(unsafe.Pointer(c))}, args...)
	result, _, _ := syscall.SyscallN(c.vtable[slot], values...)
	runtime.KeepAlive(c)
	return hresult(result)
}
func hresult(result uintptr) error {
	if int32(uint32(result)) < 0 {
		return fmt.Errorf("windows audio error 0x%08x", uint32(result))
	}
	return nil
}
func (c *comAudio) release() { _ = c.invoke(2) }

// WASAPICapture records the default playback endpoint, including all desktop
// sound. Go talks to WASAPI directly; bundled FFmpeg only encodes/muxes PCM.
type WASAPICapture struct{ FFmpeg string }
type wasapiProcess struct{ done <-chan error }

func (p *wasapiProcess) Wait() error { return <-p.done }

func (w WASAPICapture) Start(ctx context.Context, target string) (Process, error) {
	ready := make(chan error, 1)
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		started := false
		err := w.capture(ctx, target, func() { started = true; ready <- nil })
		if !started {
			ready <- err
		}
		done <- err
		close(done)
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return &wasapiProcess{done: done}, nil
}

func (w WASAPICapture) capture(parent context.Context, target string, ready func()) error {
	if err := parent.Err(); err != nil {
		return err
	}
	hr, _, _ := ole32.NewProc("CoInitializeEx").Call(0, 0)
	if err := hresult(hr); err != nil {
		return err
	}
	defer func() { _, _, _ = ole32.NewProc("CoUninitialize").Call() }()
	var enumerator *comAudio
	hr, _, _ = ole32.NewProc("CoCreateInstance").Call(uintptr(unsafe.Pointer(&clsidEnumerator)), 0, 1, uintptr(unsafe.Pointer(&iidEnumerator)), uintptr(unsafe.Pointer(&enumerator)))
	if err := hresult(hr); err != nil {
		return err
	}
	defer enumerator.release()
	var device *comAudio
	// IMMDeviceEnumerator::GetDefaultAudioEndpoint(eRender, eConsole).
	if err := enumerator.invoke(4, 0, 0, uintptr(unsafe.Pointer(&device))); err != nil {
		return err
	}
	defer device.release()
	var client *comAudio
	if err := device.invoke(3, uintptr(unsafe.Pointer(&iidAudioClient)), 1, 0, uintptr(unsafe.Pointer(&client))); err != nil {
		return err
	}
	defer client.release()
	var mix *byte
	if err := client.invoke(8, uintptr(unsafe.Pointer(&mix))); err != nil {
		return err
	}
	if mix == nil {
		return errors.New("no audio mix format")
	}
	defer func() { _, _, _ = ole32.NewProc("CoTaskMemFree").Call(uintptr(unsafe.Pointer(mix))) }()
	raw := unsafe.Slice(mix, 18)
	if raw[0] == 0xfe && raw[1] == 0xff {
		raw = unsafe.Slice(mix, 40)
	}
	format, rate, channels, align, err := waveInput(raw)
	if err != nil {
		return err
	}
	// Shared mode with AUDCLNT_STREAMFLAGS_LOOPBACK; preserve the endpoint format.
	if err = client.invoke(3, 0, 0x20000, 1000000, 0, uintptr(unsafe.Pointer(mix)), 0); err != nil {
		return err
	}
	var capture *comAudio
	if err = client.invoke(14, uintptr(unsafe.Pointer(&iidCapture)), uintptr(unsafe.Pointer(&capture))); err != nil {
		return err
	}
	defer capture.release()
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	cmd := exec.CommandContext(ctx, w.FFmpeg, "-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", format, "-ar", strconv.Itoa(rate), "-ac", strconv.Itoa(channels), "-i", "pipe:0",
		"-vn", "-c:a", "libopus", "-application", "audio", "-fec:a", "1", "-packet_loss", "10",
		"-frame_duration", "20", "-ar", "48000", "-ac", "2", "-b:a", "96000",
		"-payload_type", "111", "-flush_packets", "1", "-f", "rtp", target)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	input, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		_ = input.Close()
		return fmt.Errorf("start bundled FFmpeg: %w", err)
	}
	encoderDone := make(chan error, 1)
	go func() { encoderDone <- cmd.Wait(); close(encoderDone) }()
	defer func() { cancel(); _ = input.Close(); <-encoderDone }()
	if err = client.invoke(10); err != nil {
		return err
	}
	defer func() { _ = client.invoke(11) }()
	ready()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err = <-encoderDone:
			if err == nil {
				err = errors.New("audio encoder stopped")
			}
			return err
		case <-ticker.C:
		}
		for {
			if err = ctx.Err(); err != nil {
				return err
			}
			var frames uint32
			if err = capture.invoke(5, uintptr(unsafe.Pointer(&frames))); err != nil {
				return err
			}
			if frames == 0 {
				break
			}
			var buffer *byte
			var flags uint32
			if err = capture.invoke(3, uintptr(unsafe.Pointer(&buffer)), uintptr(unsafe.Pointer(&frames)), uintptr(unsafe.Pointer(&flags)), 0, 0); err != nil {
				return err
			}
			size := uint64(frames) * uint64(align)
			if size > 16<<20 || (buffer == nil && flags&2 == 0) {
				_ = capture.invoke(4, uintptr(frames))
				return errors.New("invalid Windows audio packet")
			}
			packet := make([]byte, int(size))
			if flags&2 == 0 {
				copy(packet, unsafe.Slice(buffer, int(size)))
			}
			// Release native memory on the same COM thread before potentially blocking.
			if err = capture.invoke(4, uintptr(frames)); err != nil {
				return err
			}
			if _, err = input.Write(packet); err != nil {
				return err
			}
		}
	}
}
