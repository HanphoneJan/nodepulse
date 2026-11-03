//go:build !windows

package main

// setWindowTheme is a no-op outside Windows. GTK and macOS title bars follow
// the system appearance, not the in-app theme.
func setWindowTheme(bool) {}
