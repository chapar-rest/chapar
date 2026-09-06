// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kpath

import (
	"bufio"
	"fmt"
	"io"
	stdpath "path"
	"strings"
)

func parseDB(root string, r io.Reader) (Context, error) {
	db := make(map[string][]string)
	sc := bufio.NewScanner(r)
	dir := root
	for sc.Scan() {
		txt := strings.TrimSpace(sc.Text())
		if txt == "" {
			continue
		}
		if txt[0] == '%' {
			continue
		}

		switch {
		case isDirDB(txt):
			dir = stdpath.Join(root, strings.TrimRight(txt, ":"))
		default:
			db[txt] = append(db[txt], stdpath.Join(dir, txt))
		}
	}

	err := sc.Err()
	if err != nil && err != io.EOF {
		return Context{}, fmt.Errorf("could not scan db file: %w", err)
	}

	return Context{db: db}, nil
}

func isDirDB(name string) bool {
	return strings.HasSuffix(name, ":") && (strings.HasPrefix(name, "/") ||
		strings.HasPrefix(name, "./") ||
		strings.HasPrefix(name, "../"))
}
