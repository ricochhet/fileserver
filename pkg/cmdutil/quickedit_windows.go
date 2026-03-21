package cmdutil

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	EnableQuickEditMode = 0x0040
	EnableExtendedFlags = 0x0080
)

// QuickEdit sets quick edit according to the specified value.
func QuickEdit(v bool) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	handle, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return fmt.Errorf("GetStdHandle: %w", err)
	}

	var mode uint32

	r1, _, err := getConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return fmt.Errorf("GetConsoleMode failed: %w", err)
	}

	mode |= EnableExtendedFlags

	if v {
		mode |= EnableQuickEditMode
	} else {
		mode &^= EnableQuickEditMode
	}

	r1, _, err = setConsoleMode.Call(uintptr(handle), uintptr(mode))
	if r1 == 0 {
		return fmt.Errorf("SetConsoleMode failed: %w", err)
	}

	return nil
}
