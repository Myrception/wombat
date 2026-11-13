package app

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	wails_options "github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wombat/internal/server"
)

var (
	appName = "Wombat"
	semver  = "0.0.0-dev"
)

// Run is the main function to run the application
func Run(assetsFS embed.FS, icon []byte) int {
	appData, err := appDataLocation(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open app data directory: %v\n", err)
		return 1
	}
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog(appData, r)
			panic(r)
		}
	}()

	if isDev {
		// Example gRPC server (localhost:5001) to try Wombat against.
		go server.Serve()
	}

	app := NewApp()
	app.appData = appData

	// Dev and release builds use separate data directories, so they may
	// run side by side.
	instanceID := "wombat-grpc-client"
	if isDev {
		instanceID += ".dev"
	}

	opts := &wails_options.App{
		Title:       appName,
		Width:       1200,
		Height:      820,
		MaxWidth:    3840,
		MaxHeight:   2560,
		AssetServer: &assetserver.Options{Assets: assetsFS},
		BackgroundColour: &wails_options.RGBA{
			R: 46, // From hex #2e3440
			G: 52,
			B: 64,
			A: 255,
		},
		SingleInstanceLock: &wails_options.SingleInstanceLock{
			UniqueId:               instanceID,
			OnSecondInstanceLaunch: app.onSecondInstanceLaunch,
		},
		Linux: &linux.Options{
			Icon:        icon,
			ProgramName: "wombat",
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
	}

	if err := wails.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "app: error running app: %v\n", err)
		return 1
	}
	return 0
}

// onSecondInstanceLaunch brings the running window to the front when the user
// starts Wombat a second time.
func (a *api) onSecondInstanceLaunch(wails_options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}
