package update

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
)

func exePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// Detect works out how this copy updates: in place when its folder is
// writable, which it is not under Program Files without elevation.
func Detect() Method {
	exe, err := exePath()
	if err != nil || !dirWritable(filepath.Dir(exe)) {
		return Manual
	}
	return InApp
}

// Install replaces the running executable with the one in the release zip at
// path. Windows will not overwrite a running exe but will rename it, so it
// moves aside to Chapar.exe.old, which Cleanup removes on the next start.
func Install(_ context.Context, path string) error {
	exe, err := exePath()
	if err != nil {
		return err
	}
	dir, base := filepath.Split(exe)
	next := filepath.Join(dir, "."+base+".update")
	if err := extractExe(path, next); err != nil {
		return err
	}
	if err := swap(exe, next, exe+".old"); err != nil {
		_ = os.Remove(next)
		return err
	}
	return nil
}

// Restart starts the installed app. The caller quits right after.
func Restart() error {
	exe, err := exePath()
	if err != nil {
		return err
	}
	return exec.Command(exe).Start()
}

// Cleanup removes the executable an earlier update moved aside.
func Cleanup() {
	if exe, err := exePath(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}
