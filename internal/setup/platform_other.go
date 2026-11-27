//go:build !linux && !windows

package setup

import "afishk/internal/config"

func configurePlatform(_ *config.Config, _ func(string, string) (string, error), _ bool) error {
	_, err := config.Defaults()
	return err
}
