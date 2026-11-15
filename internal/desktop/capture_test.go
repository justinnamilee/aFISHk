package desktop

import (
	"encoding/binary"
	"image/color"
	"testing"
)

func TestRawQtFormats(t *testing.T) {
	for _, format := range []uint32{4, 5, 6, 16, 17, 18} {
		t.Run(string(rune('a'+format)), func(t *testing.T) {
			raw := make([]byte, 8)
			if format <= 6 {
				binary.NativeEndian.PutUint32(raw, 0xff123456)
			} else {
				copy(raw, []byte{0x12, 0x34, 0x56, 0xff})
			}
			img, err := decodeRaw(raw, 1, 1, 8, format)
			if err != nil {
				t.Fatal(err)
			}
			if got := color.RGBAModel.Convert(img.At(0, 0)); got != (color.RGBA{0x12, 0x34, 0x56, 0xff}) {
				t.Fatal(got)
			}
		})
	}
}
func TestRejectMalformedImage(t *testing.T) {
	for _, x := range [][4]uint32{{0, 1, 4, 4}, {1, 1, 3, 4}, {20000, 1, 80000, 4}, {1, 1, 4, 30}, {16384, 16384, 65536, 4}} {
		if checkGeometry(x[0], x[1], x[2], x[3]) == nil {
			t.Fatal(x)
		}
	}
	if _, err := decodeRaw(nil, 1, 1, 4, 4); err == nil {
		t.Fatal("accepted truncated image")
	}
}
