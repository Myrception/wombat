package app

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v2"
	wails_options "github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"wombat/internal/server"
)

var (
	appName = "Wombat"
	semver  = "0.0.0-dev"
	appCtx  context.Context
)

// Run is the main function to run the application
func Run(js string, css string, assetsFS embed.FS) int {
	appData, err := appDataLocation(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open add data directory: %v\n", err)
		return 1
	}
	defer crashlog(appData)

	assets := &assetserver.Options{
		Assets: assetsFS,
	}

	if os.Getenv("WAILS_ENV") != "production" {
		go server.Serve()
	}

	app := NewApp()

	app.appData = appData

	opts := &wails_options.App{
		Title:       appName,
		Width:       1200,
		Height:      820,
		AssetServer: assets,
		BackgroundColour: &wails_options.RGBA{
			R: 46, // From hex #2e3440
			G: 52,
			B: 64,
			A: 255,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
	}

	err = wails.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "app: error running app: %v\n", err)
		return 1
	}
	return 0
}

func crashlog(appData string) {
	if os.Getenv("WAILS_ENV") == "production" {
		if r := recover(); r != nil {
			if _, err := os.Stat(appData); os.IsNotExist(err) {
				os.MkdirAll(appData, 0700)
			}

			var b bytes.Buffer
			b.WriteString(fmt.Sprintf("%+v\n\n", r))
			buf := make([]byte, 1<<20)
			s := runtime.Stack(buf, true)
			b.Write(buf[0:s])

			crashLogPath := filepath.Join(appData, "crash.log")
			os.WriteFile(crashLogPath, b.Bytes(), 0644)
		}
	}
}
