package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/funfunpayer/SamNPlayer/logging"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := logging.Init(); err != nil {
		println("Logging konnte nicht initialisiert werden:", err.Error())
	}
	defer logging.Close()

	app := NewApp()
	logging.SetLevel(parseLogLevel(app.settings.GetString(prefLogLevel, "info")))

	err := wails.Run(&options.App{
		Title:  "SamNPlayer",
		Width:  1000,
		Height: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 26, B: 32, A: 1},
		// Dateien per Drag & Drop annehmen. DisableWebViewDrop verhindert,
		// dass die Webview eine versehentlich fallengelassene Datei selbst
		// öffnet und damit die Oberfläche ersetzt.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
