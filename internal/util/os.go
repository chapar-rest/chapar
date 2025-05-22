package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func MakeDir(dir string) error {
	dir = filepath.FromSlash(dir)
	fnMakeDir := func() error { return os.MkdirAll(dir, os.ModePerm) }
	info, err := os.Stat(dir)
	switch {
	case err == nil:
		if info.IsDir() {
			return nil // The directory exists
		} else {
			return fmt.Errorf("path exists but is not a directory: %s", dir)
		}
	case os.IsNotExist(err):
		return fnMakeDir()
	default:
		return err
	}
}
