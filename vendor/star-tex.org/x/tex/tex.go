// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tex provides tools to typeset TeX documents.
package tex // import "star-tex.org/x/tex"

import "io"

// Processor is the interface that wraps the Process method.
type Processor interface {
	Process(w io.Writer, r io.Reader) error
}
