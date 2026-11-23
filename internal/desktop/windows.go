//go:build windows

package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var gdi32 = windows.NewLazySystemDLL("gdi32.dll")
var wtsapi = windows.NewLazySystemDLL("wtsapi32.dll")
var foreground = user32.NewProc("GetForegroundWindow")
var sendInput = user32.NewProc("SendInput")
var asyncKey = user32.NewProc("GetAsyncKeyState")

//go:uintptrescapes
func call(p *windows.LazyProc, args ...uintptr) uintptr { r, _, _ := p.Call(args...); return r }

// NativeWindows integrates only with the current, unlocked interactive session.
// No window activation, desktop capture fallback or privilege elevation is used.
type NativeWindows struct{}

func OpenWindows() (*NativeWindows, error) {
	if windows.GetCurrentProcessToken().IsElevated() {
		return nil, errors.New("run aFISHk as your normal user, without administrator elevation")
	}
	// These ABI layouts and the shipped installer target Windows 11 x64.
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return nil, errors.New("requires a 64-bit Windows build")
	}
	return &NativeWindows{}, nil
}

func interactiveDesktop() error {
	var session uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &session); err != nil || session == 0 {
		return errors.New("aFISHk must run in a signed-in desktop session")
	}
	var state *uint32
	var size uint32
	if call(wtsapi.NewProc("WTSQuerySessionInformationW"), 0, uintptr(session), 8, uintptr(unsafe.Pointer(&state)), uintptr(unsafe.Pointer(&size))) == 0 {
		return errors.New("cannot verify interactive session state")
	}
	defer call(wtsapi.NewProc("WTSFreeMemory"), uintptr(unsafe.Pointer(state)))
	if state == nil || size < 4 || *state != 0 {
		return errors.New("desktop session is disconnected or unavailable")
	}
	desk := call(user32.NewProc("OpenInputDesktop"), 0, 0, 1)
	if desk == 0 {
		return errors.New("desktop is locked, secure or unavailable")
	}
	defer call(user32.NewProc("CloseDesktop"), desk)
	var name [256]uint16
	var needed uint32
	if call(user32.NewProc("GetUserObjectInformationW"), desk, 2, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Sizeof(name)), uintptr(unsafe.Pointer(&needed))) == 0 || !strings.EqualFold(windows.UTF16ToString(name[:]), "Default") {
		return errors.New("cannot verify unlocked default desktop")
	}
	return nil
}

func (n *NativeWindows) Active(ctx context.Context) (Window, error) {
	if err := ctx.Err(); err != nil {
		return Window{}, err
	}
	if err := interactiveDesktop(); err != nil {
		return Window{}, err
	}
	hwnd := call(foreground)
	if hwnd == 0 || call(user32.NewProc("IsWindowVisible"), hwnd) == 0 || call(user32.NewProc("IsIconic"), hwnd) != 0 {
		return Window{}, errors.New("no visible focused game window")
	}
	var pid uint32
	if call(user32.NewProc("GetWindowThreadProcessId"), hwnd, uintptr(unsafe.Pointer(&pid))) == 0 || pid == 0 {
		return Window{}, errors.New("cannot identify focused process")
	}
	proc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return Window{}, fmt.Errorf("inspect focused process: %w", err)
	}
	defer func() { _ = windows.CloseHandle(proc) }()
	var exe [32768]uint16
	length := uint32(len(exe))
	if err = windows.QueryFullProcessImageName(proc, 0, &exe[0], &length); err != nil {
		return Window{}, err
	}
	var created, exited, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(proc, &created, &exited, &kernel, &user); err != nil {
		return Window{}, err
	}
	var class [256]uint16
	var title [1024]uint16
	classLen := call(user32.NewProc("GetClassNameW"), hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	titleLen := call(user32.NewProc("GetWindowTextW"), hwnd, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
	if classLen == 0 || classLen >= uintptr(len(class)-1) || titleLen == 0 || titleLen >= uintptr(len(title)-1) {
		return Window{}, errors.New("cannot read exact window identity")
	}
	if call(foreground) != hwnd {
		return Window{}, errors.New("focus changed during inspection")
	}
	return Window{ID: fmt.Sprintf("%x:%d:%08x%08x", hwnd, pid, created.HighDateTime, created.LowDateTime), Class: windows.UTF16ToString(class[:]), Title: windows.UTF16ToString(title[:]), Executable: windows.UTF16ToString(exe[:length])}, nil
}

// INPUT's union is 32 bytes, aligned to 8 bytes on Windows x64.
type keyboardInput struct {
	Kind      uint32
	Pad       uint32
	VK        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	Pad2      uint32
	Extra     uintptr
	UnionTail [8]byte
}

func (n *NativeWindows) SendF(ctx context.Context, target Window) error {
	current, err := n.Active(ctx)
	if err != nil {
		return err
	}
	if current != target {
		return errors.New("game focus or identity changed")
	}
	// Do not turn an explicit F tap into a modified shortcut or disturb a held F.
	for _, vk := range []uintptr{0x10, 0x11, 0x12, 0x5b, 0x5c, 0x46} {
		if call(asyncKey, vk)&0x8000 != 0 {
			return errors.New("release modifiers and F before tapping")
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// Recheck after all preflight work, immediately before injecting the pair.
	if current, err = n.Active(ctx); err != nil || current != target {
		return errors.New("game is no longer the verified foreground window")
	}
	events := [2]keyboardInput{{Kind: 1, VK: 0x46}, {Kind: 1, VK: 0x46, Flags: 2}}
	sent := call(sendInput, 2, uintptr(unsafe.Pointer(&events[0])), unsafe.Sizeof(events[0]))
	if sent != 2 {
		if sent == 1 {
			_ = call(sendInput, 1, uintptr(unsafe.Pointer(&events[1])), unsafe.Sizeof(events[1]))
		}
		return errors.New("input was rejected by Windows; run WoW and aFISHk as the same non-administrator user")
	}
	return nil
}

func (n *NativeWindows) Capture(ctx context.Context, id string) (image.Image, error) {
	target, err := n.Active(ctx)
	if err != nil {
		return nil, err
	}
	if target.ID != id {
		return nil, errors.New("capture target changed")
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	// PrintWindow may block inside the target. A disposable worker gives it a
	// hard deadline without leaking a goroutine or native graphics handles.
	cmd := exec.CommandContext(ctx, exe, "__capture-window")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.New("window capture failed or timed out; use WoW windowed/borderless mode")
	}
	return png.Decode(bytes.NewReader(output))
}

// CaptureWorker is an internal subprocess entry point, not an HTTP action.
func CaptureWorker(in io.Reader, out io.Writer) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var target Window
	dec := json.NewDecoder(io.LimitReader(in, 65536))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&target); err != nil {
		return err
	}
	n := &NativeWindows{}
	actual, err := n.Active(context.Background())
	if err != nil || actual != target {
		return errors.New("capture target not verified")
	}
	handle, err := strconv.ParseUint(strings.Split(target.ID, ":")[0], 16, 64)
	if err != nil {
		return err
	}
	hwnd := uintptr(handle)
	// Per-monitor physical coordinates prevent scaling/cropping on high-DPI screens.
	if call(user32.NewProc("SetThreadDpiAwarenessContext"), ^uintptr(3)) == 0 { // PER_MONITOR_AWARE_V2 (-4)
		return errors.New("cannot establish physical window coordinates")
	}
	var rect struct{ Left, Top, Right, Bottom int32 }
	if call(user32.NewProc("GetWindowRect"), hwnd, uintptr(unsafe.Pointer(&rect))) == 0 {
		return errors.New("window bounds unavailable")
	}
	width, height := int(rect.Right-rect.Left), int(rect.Bottom-rect.Top)
	if width <= 0 || height <= 0 || width > 16384 || height > 16384 || int64(width)*int64(height)*4 > 128<<20 {
		return errors.New("unsupported window dimensions")
	}
	dc := call(user32.NewProc("GetWindowDC"), hwnd)
	if dc == 0 {
		return errors.New("window DC unavailable")
	}
	defer call(user32.NewProc("ReleaseDC"), hwnd, dc)
	mem := call(gdi32.NewProc("CreateCompatibleDC"), dc)
	if mem == 0 {
		return errors.New("capture DC unavailable")
	}
	defer call(gdi32.NewProc("DeleteDC"), mem)
	info := struct {
		Size                   uint32
		Width, Height          int32
		Planes, Bits           uint16
		Compression, ImageSize uint32
		X, Y                   int32
		Used, Important        uint32
	}{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, Bits: 32}
	var pixels *byte
	bitmap := call(gdi32.NewProc("CreateDIBSection"), mem, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 || pixels == nil {
		return errors.New("capture bitmap unavailable")
	}
	defer call(gdi32.NewProc("DeleteObject"), bitmap)
	previous := call(gdi32.NewProc("SelectObject"), mem, bitmap)
	if previous == 0 || previous == ^uintptr(0) {
		return errors.New("cannot select capture bitmap")
	}
	defer call(gdi32.NewProc("SelectObject"), mem, previous)
	if call(user32.NewProc("PrintWindow"), hwnd, mem, 2) == 0 {
		return errors.New("window did not provide a screenshot")
	}
	if call(gdi32.NewProc("GdiFlush")) == 0 {
		return errors.New("cannot synchronize capture bitmap")
	}
	if actual, err = n.Active(context.Background()); err != nil || actual != target {
		return errors.New("focus changed during capture")
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	raw := unsafe.Slice(pixels, width*height*4)
	nonblack := false
	for i := 0; i < len(raw); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = raw[i+2], raw[i+1], raw[i], 255
		nonblack = nonblack || raw[i] != 0 || raw[i+1] != 0 || raw[i+2] != 0
	}
	if !nonblack {
		return errors.New("window returned a blank capture; use windowed/borderless mode")
	}
	return png.Encode(out, img)
}
