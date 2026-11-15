package desktop

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

func (k *KWin) Capture(ctx context.Context, id string) (image.Image, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		if err = r.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
	}
	stop := context.AfterFunc(ctx, func() { _ = r.Close() })
	defer stop()
	var meta map[string]dbus.Variant
	options := map[string]dbus.Variant{"include-decoration": dbus.MakeVariant(false), "include-cursor": dbus.MakeVariant(false), "native-resolution": dbus.MakeVariant(true)}
	err = k.conn.Object("org.kde.KWin.ScreenShot2", "/org/kde/KWin/ScreenShot2").CallWithContext(ctx,
		"org.kde.KWin.ScreenShot2.CaptureWindow", 0, id, options, dbus.UnixFD(w.Fd())).Store(&meta)
	if err != nil {
		return nil, fmt.Errorf("KWin window capture (check desktop entry authorization): %w", err)
	}
	_ = w.Close()
	if v, ok := meta["windowId"]; ok && v.Value() != id {
		return nil, errors.New("KWin returned a different window")
	}
	width, ok1 := meta["width"].Value().(uint32)
	height, ok2 := meta["height"].Value().(uint32)
	stride, ok3 := meta["stride"].Value().(uint32)
	format, ok4 := meta["format"].Value().(uint32)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return nil, errors.New("invalid KWin screenshot metadata")
	}
	if err = checkGeometry(width, height, stride, format); err != nil {
		return nil, err
	}
	raw := make([]byte, int(stride)*int(height))
	if _, err = io.ReadFull(r, raw); err != nil {
		return nil, fmt.Errorf("incomplete KWin image: %w", err)
	}
	return decodeRaw(raw, width, height, stride, format)
}

func checkGeometry(w, h, stride, format uint32) error {
	if w == 0 || h == 0 || w > 16384 || h > 16384 || uint64(stride)*uint64(h) > 128<<20 || uint64(stride) < uint64(w)*4 {
		return errors.New("unsupported screenshot dimensions")
	}
	// Qt QImage RGB32, ARGB32, ARGB32_Premultiplied, RGBX8888,
	// RGBA8888, RGBA8888_Premultiplied. Other/HDR formats fail explicitly.
	switch format {
	case 4, 5, 6, 16, 17, 18:
		return nil
	default:
		return fmt.Errorf("unsupported Qt pixel format %d; use SDR output", format)
	}
}
func decodeRaw(raw []byte, w, h, stride, format uint32) (image.Image, error) {
	if err := checkGeometry(w, h, stride, format); err != nil {
		return nil, err
	}
	if len(raw) != int(stride)*int(h) {
		return nil, errors.New("invalid raw image length")
	}
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			p := raw[y*int(stride)+x*4:]
			i := y*img.Stride + x*4
			var red, green, blue, alpha byte
			if format <= 6 {
				v := binary.NativeEndian.Uint32(p)
				red, green, blue, alpha = byte(v>>16), byte(v>>8), byte(v), byte(v>>24)
			} else {
				red, green, blue, alpha = p[0], p[1], p[2], p[3]
			}
			if format == 4 || format == 16 {
				alpha = 255
			}
			if format == 5 || format == 17 {
				red = byte(uint16(red) * uint16(alpha) / 255)
				green = byte(uint16(green) * uint16(alpha) / 255)
				blue = byte(uint16(blue) * uint16(alpha) / 255)
			}
			copy(img.Pix[i:i+4], []byte{red, green, blue, alpha})
		}
	}
	return img, nil
}
