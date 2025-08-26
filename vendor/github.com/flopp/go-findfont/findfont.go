// Copyright 2016 Florian Pigorsch. All rights reserved.
//
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package findfont

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Find tries to locate the specified font file in the current directory as
// well as in platform specific user and system font directories; if there is
// no exact match, Find tries substring matching.
func Find(fileName string) (filePath string, err error) {
	// check if fileName already points to a readable file
	if _, err := os.Stat(fileName); err == nil {
		return fileName, nil
	}

	// search in user and system directories
	return find(filepath.Base(fileName))
}

// List returns a list of all font files found on the system.
func List() (filePaths []string) {
	pathList := []string{}

	walkF := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if info.IsDir() == false && isFontFile(path) {
				pathList = append(pathList, path)
			}
		}
		return nil
	}
	for _, dir := range getFontDirectories() {
		filepath.Walk(dir, walkF)
	}

	return pathList
}

func isFontFile(fileName string) bool {
	lower := strings.ToLower(fileName)
	return strings.HasSuffix(lower, ".ttf") || strings.HasSuffix(lower, ".ttc") || strings.HasSuffix(lower, ".otf")
}

func stripExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func expandUser(path string) (expandedPath string) {
	if strings.HasPrefix(path, "~") {
		if u, err := user.Current(); err == nil {
			return strings.Replace(path, "~", u.HomeDir, -1)
		}
	}
	return path
}

func find(needle string) (filePath string, err error) {
	lowerNeedle := strings.ToLower(needle)
	lowerNeedleBase := stripExtension(lowerNeedle)

	match := ""
	partial := ""
	partialScore := -1

	walkF := func(path string, info os.FileInfo, err error) error {
		// we have already found a match -> nothing to do
		if match != "" {
			return nil
		}
		if err != nil {
			return nil
		}

		lowerPath := strings.ToLower(info.Name())

		if info.IsDir() == false && isFontFile(lowerPath) {
			lowerBase := stripExtension(lowerPath)
			if lowerPath == lowerNeedle {
				// exact match
				match = path
			} else if strings.Contains(lowerBase, lowerNeedleBase) {
				// partial match
				score := len(lowerBase) - len(lowerNeedle)
				if partialScore < 0 || score < partialScore {
					partialScore = score
					partial = path
				}
			}
		}
		return nil
	}

	for _, dir := range getFontDirectories() {
		filepath.Walk(dir, walkF)
		if match != "" {
			return match, nil
		}
	}

	if partial != "" {
		return partial, nil
	}

	return "", fmt.Errorf("cannot find font '%s' in user or system directories", needle)
}
