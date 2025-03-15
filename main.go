package main

import (
	"embed"
	_ "embed"
	"os"
	"wombat/internal/app"
)

//go:embed all:frontend/public
var assets embed.FS

func main() {
	os.Exit(app.Run(assets))
}
