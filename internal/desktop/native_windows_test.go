package desktop

import (
	"testing"
	"unsafe"
)

func TestWindowsInputABI(t *testing.T) {
	var input keyboardInput
	if unsafe.Sizeof(input) != 40 || unsafe.Offsetof(input.VK) != 8 || unsafe.Offsetof(input.Extra) != 24 {
		t.Fatal("INPUT/KEYBDINPUT layout does not match Windows x64")
	}
}
