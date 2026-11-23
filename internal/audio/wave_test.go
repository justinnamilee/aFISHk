package audio

import (
	"encoding/binary"
	"testing"
)

func TestWaveInput(t *testing.T) {
	for _, test := range []struct {
		tag, bits uint16
		want      string
	}{{1, 16, "s16le"}, {1, 24, "s24le"}, {1, 32, "s32le"}, {3, 32, "f32le"}, {3, 64, "f64le"}, {1, 8, ""}, {6, 32, ""}} {
		raw := make([]byte, 40)
		binary.LittleEndian.PutUint16(raw, test.tag)
		binary.LittleEndian.PutUint16(raw[2:], 2)
		binary.LittleEndian.PutUint32(raw[4:], 48000)
		binary.LittleEndian.PutUint16(raw[12:], 2*test.bits/8)
		binary.LittleEndian.PutUint16(raw[14:], test.bits)
		format, rate, channels, _, err := waveInput(raw)
		if test.want == "" {
			if err == nil {
				t.Fatal("unsupported format accepted")
			}
			continue
		}
		if err != nil || format != test.want || rate != 48000 || channels != 2 {
			t.Fatalf("%s %v", format, err)
		}
		binary.LittleEndian.PutUint16(raw, 0xfffe)
		binary.LittleEndian.PutUint16(raw[16:], 22)
		copy(raw[24:], []byte{byte(test.tag), 0, 0, 0, 0, 0, 0x10, 0, 0x80, 0, 0, 0xaa, 0, 0x38, 0x9b, 0x71})
		if format, _, _, _, err = waveInput(raw); err != nil || format != test.want {
			t.Fatal("extensible format", err)
		}
		raw[30] = 0xff
		if _, _, _, _, err = waveInput(raw); err == nil {
			t.Fatal("invalid GUID accepted")
		}
	}
	for _, raw := range [][]byte{nil, make([]byte, 17), make([]byte, 40)} {
		if _, _, _, _, err := waveInput(raw); err == nil {
			t.Fatal("invalid format accepted")
		}
	}
}
