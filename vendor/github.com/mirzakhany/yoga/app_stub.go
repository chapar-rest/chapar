//go:build nogpu

package yoga

import (
	"errors"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// Window is the no-GPU stub. The headless build (-tags nogpu) does not link the
// windowing/WebGPU stack, so New always fails; the type exists only so the
// package compiles and the public API is consistent across build tags.
type Window struct{}

// New always returns an error under the nogpu build tag.
func New(Config) (*Window, error) {
	return nil, errors.New("yoga: GPU build required (built with -tags nogpu)")
}

// Atlas is a no-op in the headless build.
func (a *Window) Atlas() *render.FontAtlas { return nil }

// Text is a no-op in the headless build; use yoga.Text() after SetResources.
func (a *Window) Text() *shape.Engine { return nil }

// Clipboard is a no-op in the headless build.
func (a *Window) Clipboard() input.Clipboard { return nil }

// runApp is a no-op in the headless build; yoga.Run returns New's error before
// reaching it (the nogpu build never constructs a Window).
func (a *Window) runApp(App) {}

// Close is a no-op in the headless build.
func (a *Window) Close() {}
