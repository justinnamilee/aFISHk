// Command afishk runs the desktop companion under the logged-in desktop user.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"afishk/internal/audio"
	"afishk/internal/browser"
	"afishk/internal/config"
	"afishk/internal/desktop"
	"afishk/internal/platform"
	"afishk/internal/server"
	"afishk/internal/setup"
)

var version = "dev"

func main() {
	prepareArgs()
	if err := run(); err != nil {
		reportError(err)
		os.Exit(1)
	}
}
func run() error {
	if handled, err := nativeCommand(); handled {
		return err
	}
	if len(os.Args) > 1 && os.Args[1] == "launch" {
		return desktopLaunch(os.Args[2:])
	}
	if len(os.Args) > 1 && (os.Args[1] == "setup" || os.Args[1] == "firstrun") {
		args := flag.NewFlagSet("setup", flag.ContinueOnError)
		path := args.String("config", setup.DefaultPath(), "new configuration file (never overwrites)")
		web := args.Bool("web", false, "open local browser setup")
		if err := args.Parse(os.Args[2:]); err != nil {
			return err
		}
		if args.NArg() != 0 {
			return errors.New("usage: afishk setup [--web] [--config PATH]")
		}
		if *web {
			return webSetup(context.Background(), *path)
		}
		return setup.Run(os.Stdin, os.Stdout, *path)
	}
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("afishk", version)
		return nil
	}
	if len(os.Args) > 1 && os.Args[1] == "inspect" {
		k, err := platform.Open(config.Config{})
		if err != nil {
			return err
		}
		defer k.Close()
		fmt.Fprintln(os.Stderr, "Focus WoW now; querying active window in 5 seconds…")
		time.Sleep(5 * time.Second)
		w, err := k.Windows.Active(context.Background())
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(w)
	}
	path := flag.String("config", setup.DefaultPath(), "configuration file")
	check := flag.Bool("check", false, "validate configuration without starting")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unknown command %q; use setup, inspect, version, or --help", flag.Arg(0))
	}
	c, err := config.Load(*path)
	if err != nil {
		return err
	}
	if *check {
		fmt.Println("Configuration valid")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serve(ctx, c, nil)
}

func webSetup(ctx context.Context, path string) error {
	b, err := platform.Open(config.Config{})
	if err != nil {
		return err
	}
	defer b.Close()
	return setup.RunWeb(ctx, path, b.Windows, browser.Open, os.Stdout)
}

func serve(ctx context.Context, c config.Config, ready func()) error {
	if c.AllowUnauthenticatedLAN && c.TokenFile == "" {
		log.Print("WARNING: authentication disabled; restrict listener/firewall to trusted VPN clients")
	}
	k, err := platform.Open(c)
	if err != nil {
		return err
	}
	defer k.Close()
	control := &desktop.Controller{FishKey: c.FishKey, Windows: k.Windows, Capture: k.Capture, Input: k.Input, Class: c.GameClass, Title: c.GameTitle, CacheDir: c.CacheDir, Executable: c.GameExecutable}
	a := audio.New(c, k.Audio)
	defer a.Close()
	app, err := server.New(c, control, a)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: c.Listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 45 * time.Second, MaxHeaderBytes: 16 << 10}

	shutdownDone := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdown); err != nil {
				_ = srv.Close()
			}
		case <-done:
		}
	}()
	ln, err := net.Listen("tcp", c.Listen)
	if err != nil {
		close(done)
		<-shutdownDone
		return err
	}
	defer func() { _ = ln.Close() }()
	if ready != nil {
		ready()
	}
	log.Printf("aFISHk %s listening on %s", version, c.Listen)
	if c.TLSCert != "" {
		err = srv.ServeTLS(ln, c.TLSCert, c.TLSKey)
	} else {
		err = srv.Serve(ln)
	}
	close(done)
	<-shutdownDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
