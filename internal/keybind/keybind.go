// Package keybind resolves local configuration into physical keyboard keys.
package keybind

import (
	"fmt"
	"strings"
)

type Key struct {
	Linux    int    // evdev code
	Scan     uint16 // Windows set 1 scan code
	Extended bool
}

// Standard PC keyboard positions, named after the browser's KeyboardEvent.code
// with Key/Digit prefixes omitted for letters and the number row.
var keys = func() map[string]Key {
	m := map[string]Key{}
	row := func(first int, names string) {
		for i, name := range strings.Fields(names) {
			n := first + i
			m[strings.ToUpper(name)] = Key{n, uint16(n), false}
		}
	}
	row(1, "Escape 1 2 3 4 5 6 7 8 9 0 Minus Equal Backspace Tab Q W E R T Y U I O P BracketLeft BracketRight Enter")
	row(30, "A S D F G H J K L Semicolon Quote Backquote")
	row(43, "Backslash Z X C V B N M Comma Period Slash")
	row(55, "NumpadMultiply")
	row(57, "Space")
	row(59, "F1 F2 F3 F4 F5 F6 F7 F8 F9 F10")
	row(71, "Numpad7 Numpad8 Numpad9 NumpadSubtract Numpad4 Numpad5 Numpad6 NumpadAdd Numpad1 Numpad2 Numpad3 Numpad0 NumpadDecimal")
	row(86, "IntlBackslash F11 F12")
	for name, k := range map[string]Key{
		"NUMPADENTER": {96, 28, true}, "NUMPADDIVIDE": {98, 53, true},
		"HOME": {102, 71, true}, "ARROWUP": {103, 72, true}, "PAGEUP": {104, 73, true},
		"ARROWLEFT": {105, 75, true}, "ARROWRIGHT": {106, 77, true}, "END": {107, 79, true},
		"ARROWDOWN": {108, 80, true}, "PAGEDOWN": {109, 81, true}, "INSERT": {110, 82, true}, "DELETE": {111, 83, true},
	} {
		m[name] = k
	}
	return m
}()

// Parse returns keys in press order. Callers release them in reverse.
func Parse(binding string) ([]Key, error) {
	parts := strings.Split(strings.ToUpper(binding), "+")
	ctrl, shift := false, false
	for _, mod := range parts[:len(parts)-1] {
		switch strings.TrimSpace(mod) {
		case "CTRL":
			if ctrl {
				return nil, fmt.Errorf("duplicate Ctrl in %q", binding)
			}
			ctrl = true
		case "SHIFT":
			if shift {
				return nil, fmt.Errorf("duplicate Shift in %q", binding)
			}
			shift = true
		default:
			return nil, fmt.Errorf("use only Ctrl or Shift before the key in %q", binding)
		}
	}
	base, ok := keys[strings.TrimSpace(parts[len(parts)-1])]
	if !ok {
		return nil, fmt.Errorf("unsupported key in %q", binding)
	}
	var chord []Key
	if ctrl {
		chord = append(chord, Key{29, 29, false})
	}
	if shift {
		chord = append(chord, Key{42, 42, false})
	}
	return append(chord, base), nil
}
