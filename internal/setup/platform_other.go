//go:build !linux

package setup

import "afishk/internal/config"

func configurePlatform(_ *config.Config, _ func(string, string) (string, error)) error {
	_, err := config.Defaults()
	return err
}
