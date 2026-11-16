package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"afishk/internal/browser"
	"afishk/internal/config"
	"afishk/internal/desktop"
	"afishk/internal/privatefs"
	"afishk/internal/setup"
	"golang.org/x/sys/windows"
)

func prepareArgs() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "launch")
	}
}
func reportError(err error) {
	log.Print(err)
	if len(os.Args) > 1 && os.Args[1] == "__capture-window" {
		return
	}
	for _, arg := range os.Args {
		if arg == "--background" {
			return
		}
	}
	text, _ := windows.UTF16PtrFromString(err.Error())
	title, _ := windows.UTF16PtrFromString("aFISHk")
	_, _, _ = windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10)
}
func objectName(path, suffix string) *uint16 {
	abs, _ := filepath.Abs(path)
	sum := sha256.Sum256([]byte(strings.ToLower(abs)))
	name, _ := windows.UTF16PtrFromString("Local\\aFISHk-" + hex.EncodeToString(sum[:16]) + suffix)
	return name
}
func nativeCommand() (bool, error) {
	if len(os.Args) < 2 {
		return false, nil
	}
	switch os.Args[1] {
	case "__capture-window":
		return true, desktop.CaptureWorker(os.Stdin, os.Stdout)
	case "stop":
		args := flag.NewFlagSet("stop", flag.ContinueOnError)
		path := args.String("config", setup.DefaultPath(), "configuration path")
		if err := args.Parse(os.Args[2:]); err != nil {
			return true, err
		}
		if args.NArg() != 0 {
			return true, errors.New("usage: afishk stop [--config PATH]")
		}
		return true, stopDesktop(*path)
	case "open":
		c, err := config.Load(setup.DefaultPath())
		if err != nil {
			return true, err
		}
		return true, browser.Open(c.PublicOrigin)
	}
	return false, nil
}
func stopDesktop(path string) error {
	event, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, objectName(path, "-stop"))
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(event) }()
	if err = windows.SetEvent(event); err != nil {
		return err
	}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		handle, e := windows.OpenMutex(windows.SYNCHRONIZE, false, objectName(path, "-instance"))
		if errors.Is(e, windows.ERROR_FILE_NOT_FOUND) {
			return nil
		}
		if e != nil {
			return e
		}
		_ = windows.CloseHandle(handle)
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("aFISHk did not stop; close it in Task Manager before upgrading")
}
func desktopLaunch(argv []string) (launchErr error) {
	args := flag.NewFlagSet("launch", flag.ContinueOnError)
	path := args.String("config", setup.DefaultPath(), "configuration path")
	background := args.Bool("background", false, "start at sign-in without opening a browser")
	if err := args.Parse(argv); err != nil {
		return err
	}
	if args.NArg() != 0 {
		return errors.New("usage: afishk launch [--background] [--config PATH]")
	}
	if version := windows.RtlGetVersion(); version.MajorVersion < 10 || version.BuildNumber < 22000 {
		return errors.New("requires Windows 11 or later")
	}
	mutex, err := windows.CreateMutex(nil, false, objectName(*path, "-instance"))
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if mutex != 0 {
			_ = windows.CloseHandle(mutex)
		}
		if !*background {
			if c, e := config.Load(*path); e == nil {
				return browser.Open(c.PublicOrigin)
			}
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(mutex) }()
	event, err := windows.CreateEvent(nil, 1, 0, objectName(*path, "-stop"))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(event) }()
	if err = windows.ResetEvent(event); err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cache = filepath.Join(cache, "afishk")
	if err = privatefs.Dir(cache); err != nil {
		return err
	}
	logPath := filepath.Join(cache, "afishk.log")
	if stat, e := os.Stat(logPath); e == nil && stat.Size() > 1<<20 {
		_ = os.Remove(logPath + ".old")
		_ = os.Rename(logPath, logPath+".old")
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()
	if err = privatefs.CheckFile(logPath); err != nil {
		return err
	}
	log.SetOutput(logFile)
	defer log.SetOutput(os.Stderr)
	defer func() {
		if launchErr != nil {
			log.Print(launchErr)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			result, e := windows.WaitForSingleObject(event, 250)
			if e != nil || result == windows.WAIT_OBJECT_0 {
				cancel()
				return
			}
		}
	}()
	defer func() { cancel(); <-watcherDone }()
	firstRun := false
	if _, e := os.Stat(*path); os.IsNotExist(e) {
		if *background {
			return errors.New("open aFISHk from Start Menu to finish first-run setup")
		}
		firstRun = true
		if err = webSetup(ctx, *path); err != nil {
			return err
		}
	}
	c, err := config.Load(*path)
	if err != nil {
		return err
	}
	var ready func()
	if !*background && !firstRun {
		ready = func() { _ = browser.Open(c.PublicOrigin) }
	}
	err = serve(ctx, c, ready)
	return err
}
