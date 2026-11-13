package main

import (
	"embed"
	"os"
	"wombat/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	os.Exit(app.Run(assets))
}
