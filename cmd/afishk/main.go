// Command afishk runs the desktop companion under the logged-in Plasma user.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"afishk/internal/addon"
	"afishk/internal/audio"
	"afishk/internal/config"
	"afishk/internal/desktop"
	"afishk/internal/server"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) > 1 && os.Args[1] == "install-addon" {
		args := flag.NewFlagSet("install-addon", flag.ContinueOnError)
		archive := args.String("zip", "", "downloaded FishFry ZIP")
		wow := args.String("wow-dir", "", "game-client directory containing Interface/AddOns")
		if err := args.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *archive == "" || *wow == "" {
			return errors.New("usage: afishk install-addon --zip FishFry.zip --wow-dir /path/to/game")
		}
		if err := addon.Install(*archive, *wow); err != nil {
			return err
		}
		fmt.Println("FishFry installed; enable it in WoW and bind F manually")
		return nil
	}

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("afishk", version)
		return nil
	}
	if len(os.Args) > 1 && os.Args[1] == "inspect" {
		k, err := desktop.Open()
		if err != nil {
			return err
		}
		defer k.Close()
		fmt.Fprintln(os.Stderr, "Focus WoW now; querying active window in 5 seconds…")
		time.Sleep(5 * time.Second)
		w, err := k.Active(context.Background())
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(w)
	}
	path := flag.String("config", "config.json", "configuration file")
	check := flag.Bool("check", false, "validate configuration without starting")
	flag.Parse()
	c, err := config.Load(*path)
	if err != nil {
		return err
	}
	if *check {
		fmt.Println("Configuration valid")
		return nil
	}
	if c.AllowUnauthenticatedLAN && c.TokenFile == "" {
		log.Print("WARNING: authentication disabled; restrict listener/firewall to trusted VPN clients")
	}
	k, err := desktop.Open()
	if err != nil {
		return err
	}
	defer k.Close()
	control := &desktop.Controller{Backend: k, Class: c.GameClass, Title: c.GameTitle, Ydotool: c.Ydotool, CacheDir: c.CacheDir}
	a := audio.New(c)
	defer a.Close()
	app, err := server.New(c, control, a)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: c.Listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 45 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdown)
		case <-done:
		}
	}()
	log.Printf("aFISHk %s listening on %s", version, c.Listen)
	if c.TLSCert != "" {
		err = srv.ListenAndServeTLS(c.TLSCert, c.TLSKey)
	} else {
		err = srv.ListenAndServe()
	}
	close(done)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
