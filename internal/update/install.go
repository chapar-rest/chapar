package update

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Method is how this copy of Chapar gets a new version.
type Method int

const (
	// Manual: the user downloads and installs the new version.
	Manual Method = iota
	// InApp: Install replaces this copy in place.
	InApp
	// Homebrew: the app came from the Homebrew cask, which BrewUpgrade runs.
	Homebrew
)

// swap moves next into cur's place. cur is kept at old until next is in, so
// a failed swap puts it back.
func swap(cur, next, old string) error {
	_ = os.RemoveAll(old)
	if err := os.Rename(cur, old); err != nil {
		return err
	}
	if err := os.Rename(next, cur); err != nil {
		if rerr := os.Rename(old, cur); rerr != nil {
			return fmt.Errorf("%w (and restoring %s failed: %v)", err, cur, rerr)
		}
		return err
	}
	return nil
}

// extractExe writes the archive's executable (the only .exe in a Windows
// release zip) to dest.
func extractExe(zipPath, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()
	var exe *zip.File
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && strings.EqualFold(filepath.Ext(f.Name), ".exe") {
			if exe != nil {
				return errors.New("release archive has more than one executable")
			}
			exe = f
		}
	}
	if exe == nil {
		return errors.New("release archive has no executable")
	}
	src, err := exe.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, src)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(dest)
	}
	return err
}
