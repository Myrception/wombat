//go:build !dev

package app

// isDev is true for `wails dev` builds (the Wails CLI sets the "dev" tag).
const isDev = false
