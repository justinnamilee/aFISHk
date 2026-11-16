package config

import "errors"

func platformDefaults(c *Config) error {
	c.Ydotool = "ydotool"
	c.FFmpeg = "ffmpeg"
	c.PulseSource = "@DEFAULT_MONITOR@"
	return nil
}

func validatePlatform(c Config) error {
	if c.FFmpeg == "" || c.Ydotool == "" || c.PulseSource == "" {
		return errors.New("ffmpeg, ydotool and pulse_source must be nonempty")
	}
	return nil
}
