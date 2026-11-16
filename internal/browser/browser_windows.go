package browser

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

func Open(url string) error {
	action, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	r, _, _ := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW").Call(0, uintptr(unsafe.Pointer(action)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("cannot open default browser (%d)", r)
	}
	return nil
}
