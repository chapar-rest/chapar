// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kpath

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

func parseConfig(r io.Reader) (Context, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		raw := bytes.TrimSpace(sc.Bytes())
		if len(raw) == 0 {
			continue
		}
		if raw[0] == '%' {
			continue
		}
	}

	err := sc.Err()
	if err != nil && err != io.EOF {
		return Context{}, fmt.Errorf("could not scan config file: %w", err)
	}

	panic("not implemented")
}
