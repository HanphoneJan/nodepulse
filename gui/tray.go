package main

import (
	"context"
	_ "embed"

	"github.com/HanphoneJan/nodepulse/internal/autostart"
	"github.com/energye/systray"
)

//go:embed tray-icon.png
var trayIcon []byte

var trayAutostartItem *systray.MenuItem

// startTray runs the system tray in its own goroutine so the Wails main loop
// is never blocked. energye/systray talks to the StatusNotifier/AppIndicator
// DBus service on Linux instead of owning the GTK main loop.
func (a *App) startTray() {
	go systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTitle("NodePulse")
		systray.SetTooltip("NodePulse — proxy node stability")

		open := systray.AddMenuItem("打开面板", "Show the NodePulse window")
		open.Click(a.showWindow)

		autostartItem := systray.AddMenuItemCheckbox("开机启动", "Start NodePulse at login in the background", autostart.Enabled())
		trayAutostartItem = autostartItem
		autostartItem.Click(func() {
			enabled := !autostart.Enabled()
			if err := a.SetAutostart(enabled); err != nil {
				_ = sendNotification("NodePulse", "开机启动设置失败: "+err.Error())
				return
			}
		})

		systray.AddSeparator()
		quit := systray.AddMenuItem("退出", "Quit NodePulse")
		quit.Click(func() {
			systray.Quit()
			a.quitApp()
		})
	}, func() {})
}

// shutdown stops the tray loop when Wails exits for any reason (SIGTERM,
// window manager quit, or the tray menu).
func (a *App) shutdown(context.Context) {
	_ = a.StopDaemon()
	systray.Quit()
}

func setTrayAutostart(enabled bool) {
	if trayAutostartItem == nil {
		return
	}
	if enabled {
		trayAutostartItem.Check()
		return
	}
	trayAutostartItem.Uncheck()
}
