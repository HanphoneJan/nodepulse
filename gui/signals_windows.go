//go:build windows

package main

import (
	"os"
	"os/signal"

	"github.com/energye/systray"
)

func installSignalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		systray.Quit()
		os.Exit(0)
	}()
}
