//go:build !linux

package platform

import (
	"afishk/internal/config"
	"fmt"
	"runtime"
)

func Open(config.Config) (*Backends, error) {
	return nil, fmt.Errorf("desktop backends are not implemented for %s; supported: Linux KDE Plasma Wayland", runtime.GOOS)
}
