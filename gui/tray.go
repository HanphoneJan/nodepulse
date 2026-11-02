package main

import (
	_ "embed"

	"github.com/energye/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed tray-icon.png
var trayIcon []byte

// startTray runs the system tray in its own goroutine so the Wails main loop
// is never blocked. energye/systray talks to the StatusNotifier/AppIndicator
// DBus service on Linux instead of owning the GTK main loop.
func (a *App) startTray() {
	go systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTitle("NodePulse")
		systray.SetTooltip("NodePulse — proxy node stability")

		open := systray.AddMenuItem("打开面板", "Show the NodePulse window")
		open.Click(func() {
			if a.ctx != nil {
				wailsruntime.WindowShow(a.ctx)
			}
		})
		systray.AddSeparator()
		quit := systray.AddMenuItem("退出", "Quit NodePulse")
		quit.Click(func() {
			systray.Quit()
			if a.ctx != nil {
				wailsruntime.Quit(a.ctx)
			}
		})
	}, func() {})
}
