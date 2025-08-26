// +build dragonfly freebsd linux nacl netbsd openbsd solaris

// Copyright 2016 Florian Pigorsch. All rights reserved.
//
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package findfont

import (
	"os"
	"path/filepath"
)

func getFontDirectories() (paths []string) {
	directories := getUserFontDirs()
	directories = append(directories, getSystemFontDirs()...)
	return directories
}

func getUserFontDirs() (paths []string) {
	if dataPath := os.Getenv("XDG_DATA_HOME"); dataPath != "" {
		return []string{expandUser("~/.fonts/"), filepath.Join(expandUser(dataPath), "fonts")}
	}
	return []string{expandUser("~/.fonts/"), expandUser("~/.local/share/fonts/")}
}

func getSystemFontDirs() (paths []string) {
	if dataPaths := os.Getenv("XDG_DATA_DIRS"); dataPaths != "" {
		for _, dataPath := range filepath.SplitList(dataPaths) {
			paths = append(paths, filepath.Join(expandUser(dataPath), "fonts"))
		}
		return paths
	}
	return []string{"/usr/local/share/fonts/", "/usr/share/fonts/"}
}
