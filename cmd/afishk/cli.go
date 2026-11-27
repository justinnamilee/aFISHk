package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	"afishk/internal/setup"
)

func usage() {
	out := flag.CommandLine.Output()
	_, _ = fmt.Fprintln(out, `Usage: afishk [--config PATH] [--check]
       afishk COMMAND [OPTIONS]

Run without a command to start the desktop companion.

Commands:
  setup [--web] [--overwrite] [--config PATH]
      Create configuration in the terminal, or open browser setup with --web.
      Use --overwrite to save over an existing config while keeping its token.
      firstrun is an alias for setup.
  setup --rotate-token [--config PATH]
      Generate a new browser password and leave the configuration file intact.
  inspect
      Show the focused game window's identity after five seconds.
  version
      Print the installed version.`)
	if runtime.GOOS == "windows" {
		_, _ = fmt.Fprintln(out, `  launch [--background] [--config PATH]
      Start the desktop companion and open its page (the default on Windows).
      --background suppresses the browser and first-run wizard.
  stop [--config PATH]
      Stop the desktop companion.
  open
      Open the configured control page in the default browser.`)
	}
	_, _ = fmt.Fprintln(out, "\nOptions:")
	flag.PrintDefaults()
	_, _ = fmt.Fprintln(out, "  -h, -help, --help\n        show this help")
}

func helpResult(err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	return err
}

func setupCommand(argv []string) error {
	args := flag.NewFlagSet("setup", flag.ContinueOnError)
	path := args.String("config", setup.DefaultPath(), "configuration file")
	web := args.Bool("web", false, "open local browser setup")
	overwrite := args.Bool("overwrite", false, "replace existing configuration after saving, keeping its token")
	rotate := args.Bool("rotate-token", false, "generate a new browser password without changing configuration")
	if err := args.Parse(argv); err != nil {
		return helpResult(err)
	}
	if args.NArg() != 0 {
		return errors.New("unexpected setup arguments; use afishk setup --help")
	}
	if *rotate {
		if *web || *overwrite {
			return errors.New("--rotate-token must be used separately from --web and --overwrite")
		}
		path, err := setup.RotateToken(*path)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "New browser password saved to %s. Restart aFISHk to apply it.\n", path)
		return nil
	}
	if *web {
		return webSetup(context.Background(), *path, *overwrite)
	}
	return setup.Run(os.Stdin, os.Stdout, *path, *overwrite)
}
