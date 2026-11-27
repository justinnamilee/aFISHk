package setup

import "afishk/internal/config"

func configurePlatform(c *config.Config, ask func(string, string) (string, error), overwrite bool) error {
	if !overwrite {
		c.Ydotool = "/usr/bin/ydotool"
		c.FFmpeg = "/usr/bin/ffmpeg"
	}
	var err error
	c.PulseSource, err = ask("Pulse audio source", c.PulseSource)
	return err
}
