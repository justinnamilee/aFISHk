package setup

import "afishk/internal/config"

func configurePlatform(c *config.Config, ask func(string, string) (string, error), _ bool) error {
	var err error
	c.GameExecutable, err = ask("Full WoW executable path from afishk inspect", c.GameExecutable)
	return err
}
