// SPDX-License-Identifier: Unlicense OR MIT

//go:build !windows && !android && !js && !darwin && !ios && !linux
// +build !windows,!android,!js,!darwin,!ios,!linux

package explorer

import (
	"io"

	"gioui.org/app"
	"gioui.org/io/event"
)

type explorer struct{}

func newExplorer(w *app.Window) *explorer {
	return new(explorer)
}

func (e *Explorer) listenEvents(_ event.Event) {}

func (e *Explorer) exportFile(_ string) (io.WriteCloser, error) {
	return nil, ErrNotAvailable
}

func (e *Explorer) importFile(_ ...string) (io.ReadCloser, error) {
	return nil, ErrNotAvailable
}

func (e *Explorer) importFiles(_ ...string) ([]io.ReadCloser, error) {
	return nil, ErrNotAvailable
}
