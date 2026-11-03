//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	dwmapi                     = syscall.NewLazyDLL("dwmapi.dll")
	procEnumWindows            = user32.NewProc("EnumWindows")
	procGetWindowThreadProcess = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible        = user32.NewProc("IsWindowVisible")
	procDwmSetWindowAttribute  = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	dwmwaUseImmersiveDarkMode    = 20
	dwmwaUseImmersiveDarkModeOld = 19
)

// setWindowTheme switches the native Windows title bar between light and dark
// so it follows the in-app theme instead of only the OS setting.
func setWindowTheme(dark bool) {
	hwnd := findProcessWindow()
	if hwnd == 0 {
		return
	}
	value := int32(0)
	if dark {
		value = 1
	}
	if hr, _, _ := procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value)); hr != 0 {
		procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkModeOld, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
	}
}

func findProcessWindow() uintptr {
	pid := uint32(os.Getpid())
	var found uintptr
	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var owner uint32
		procGetWindowThreadProcess.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
		if owner != pid {
			return 1
		}
		if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 {
			return 1
		}
		found = hwnd
		return 0
	})
	procEnumWindows.Call(callback, 0)
	return found
}
