package main

import (
	"embed"
	_ "embed"
	"os"
	"wombat/internal/app"
)

//go:embed all:frontend/public
var assets embed.FS

//go:embed frontend/public/build/bundle.js
var js string

//go:embed frontend/public/build/bundle.css
var css string

//go:embed frontend/public/build/extra.css
var extra string

func main() {
	os.Exit(app.Run(js, css+extra, assets))
}
