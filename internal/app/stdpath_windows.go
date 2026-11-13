package app

import (
	"os"
	"path/filepath"
)

func userDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "AppData", "Roaming"), nil
}
