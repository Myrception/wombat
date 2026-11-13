package app

import (
	"os"
	"path/filepath"
)

// dataDirEnv overrides the directory Wombat stores its database in.
const dataDirEnv = "WOMBAT_DATA_DIR"

// appDataLocation returns (and creates) the directory for application data.
//
// Dev builds default to ./.data so that `wails dev` never touches, or competes
// for the database lock with, an installed Wombat.
func appDataLocation(name string) (string, error) {
	p := os.Getenv(dataDirEnv)
	if p == "" && isDev {
		p = ".data"
	}
	if p == "" {
		base, err := userDataDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(base, name)
	}
	if err := os.MkdirAll(p, 0700); err != nil {
		return "", err
	}
	return filepath.Abs(p)
}
