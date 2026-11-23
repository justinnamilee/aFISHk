package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Parse WAVEFORMATEX/WAVEFORMATEXTENSIBLE without assuming Go struct packing.
func waveInput(raw []byte) (format string, rate, channels, align int, err error) {
	bad := errors.New("unsupported Windows audio mix format")
	if len(raw) < 18 {
		return "", 0, 0, 0, bad
	}
	tag := binary.LittleEndian.Uint16(raw)
	channels = int(binary.LittleEndian.Uint16(raw[2:]))
	rate = int(binary.LittleEndian.Uint32(raw[4:]))
	align = int(binary.LittleEndian.Uint16(raw[12:]))
	bits := int(binary.LittleEndian.Uint16(raw[14:]))
	if tag == 0xfffe {
		if len(raw) < 40 || binary.LittleEndian.Uint16(raw[16:]) < 22 {
			return "", 0, 0, 0, bad
		}
		tail := []byte{0, 0, 0, 0, 0x10, 0, 0x80, 0, 0, 0xaa, 0, 0x38, 0x9b, 0x71}
		if !bytes.Equal(raw[26:40], tail) {
			return "", 0, 0, 0, bad
		}
		tag = binary.LittleEndian.Uint16(raw[24:])
	}
	if channels < 1 || channels > 32 || rate < 8000 || rate > 384000 || align != channels*bits/8 {
		return "", 0, 0, 0, bad
	}
	switch {
	case tag == 3 && (bits == 32 || bits == 64):
		format = fmt.Sprintf("f%dle", bits)
	case tag == 1 && (bits == 16 || bits == 24 || bits == 32):
		format = fmt.Sprintf("s%dle", bits)
	default:
		return "", 0, 0, 0, bad
	}
	return format, rate, channels, align, nil
}
