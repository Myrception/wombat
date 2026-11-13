package main

import (
	"embed"
	"os"

	"wombat/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	os.Exit(app.Run(assets, icon))
}
