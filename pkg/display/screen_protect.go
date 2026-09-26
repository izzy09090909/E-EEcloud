package display

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procSetWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")
	procFindWindowW              = user32.NewProc("FindWindowW")
)

const WDA_EXCLUDEFROMCAPTURE = 0x00000011

// EnableBlackoutByTitle finds the window handle by title and applies Win32 anti-screen recording
func EnableBlackoutByTitle(windowTitle string) error {
	titlePtr, err := syscall.UTF16PtrFromString(windowTitle)
	if err != nil {
		return fmt.Errorf("failed to convert window title: %w", err)
	}

	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return fmt.Errorf("window not found: %s", windowTitle)
	}

	r1, _, err := procSetWindowDisplayAffinity.Call(hwnd, uintptr(WDA_EXCLUDEFROMCAPTURE))
	if r1 == 0 {
		return fmt.Errorf("SetWindowDisplayAffinity failed: %w", err)
	}

	return nil
}