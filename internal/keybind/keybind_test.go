package keybind

import (
	"slices"
	"testing"
)

func TestChords(t *testing.T) {
	for text, want := range map[string][]Key{
		"F": {{33, 33, false}}, "shift + ctrl + F12": {{29, 29, false}, {42, 42, false}, {88, 88, false}},
		"Ctrl+ArrowUp": {{29, 29, false}, {103, 72, true}}, "NumpadEnter": {{96, 28, true}},
	} {
		got, err := Parse(text)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("Parse(%q) = %v, %v", text, got, err)
		}
	}
	for _, text := range []string{"", "Shift", "Alt+F", "Ctrl+Ctrl+F", "F+G", "Ctrl+", "Ctrl+A+B", "F99"} {
		if _, err := Parse(text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
}
