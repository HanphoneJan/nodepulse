package main

import (
	"flag"

	"github.com/HanphoneJan/nodepulse/internal/uiassets"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	background := flag.Bool("background", false, "start hidden in the system tray")
	flag.Parse()

	tuneWebKit()
	app := NewApp()
	app.startHidden = *background
	installSignalHandler()
	go app.startTray()
	err := wails.Run(&options.App{
		Title:     "NodePulse",
		Width:     1180,
		Height:    760,
		MinWidth:  920,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: uiassets.Dist(),
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 22, A: 1},
		StartHidden:      *background,
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
