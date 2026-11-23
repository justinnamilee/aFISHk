package config

import (
	"errors"
	"os"
	"path/filepath"
)

func platformDefaults(c *Config) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	c.FFmpeg = filepath.Join(filepath.Dir(exe), "ffmpeg.exe")
	return nil
}
func validatePlatform(c Config) error {
	if !filepath.IsAbs(c.FFmpeg) {
		return errors.New("ffmpeg must be an absolute path")
	}
	if !filepath.IsAbs(c.GameExecutable) {
		return errors.New("select the exact WoW executable during setup")
	}
	return nil
}
