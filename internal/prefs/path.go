package prefs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const AppName = "chapar"

// GetConfigDir returns the appropriate directory for storing configuration files
// based on the current operating system.
func GetConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%\chapar
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, AppName)

	case "darwin":
		// macOS: ~/Library/Application Support/chapar
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, "Library", "Application Support", AppName)

	case "js":
		// When running as WASM, return empty as we'll use browser storage
		return "", nil

	default:
		// Linux and others: ~/.config/chapar
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", AppName)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetConfigDirFilePath returns the full path for a configuration file with the given name
func getFilePath(fileName string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Handle the WASM case
	if configDir == "" && runtime.GOOS == "js" {
		return fileName, nil // Just return the filename for browser storage keys
	}

	return filepath.Join(configDir, fileName), nil
}
