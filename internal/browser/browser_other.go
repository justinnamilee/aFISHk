//go:build !linux && !windows

package browser

import "errors"

func Open(string) error { return errors.New("open the displayed setup URL in your browser") }
