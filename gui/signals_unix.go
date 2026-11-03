//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/energye/systray"
)

// installSignalHandler guarantees logout/TERM does not leave the tray and
// WebKit loops alive after Wails begins shutting down.
func installSignalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		systray.Quit()
		os.Exit(0)
	}()
}
