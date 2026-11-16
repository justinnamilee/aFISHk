//go:build !linux

package config

import (
	"fmt"
	"runtime"
)

func platformDefaults(*Config) error { return unsupportedPlatform() }
func validatePlatform(Config) error  { return unsupportedPlatform() }
func unsupportedPlatform() error {
	return fmt.Errorf("backends are not implemented for %s; configuration requires Linux KDE Plasma Wayland", runtime.GOOS)
}
