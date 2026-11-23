//go:build !windows

package main

import (
	"errors"
	"log"
)

func prepareArgs()                 {}
func nativeCommand() (bool, error) { return false, nil }
func reportError(err error)        { log.Print(err) }
func desktopLaunch([]string) error {
	return errors.New("launch is for Windows; use the Linux user service or run afishk directly")
}
