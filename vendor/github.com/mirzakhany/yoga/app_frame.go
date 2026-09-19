//go:build !nogpu

package yoga

import (
	"fmt"
	"strings"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// Text returns the shaped text engine for constructing widgets.
func (a *Window) Text() *shape.Engine { return a.text }

// Atlas returns the glyph atlas (legacy accessor).
func (a *Window) Atlas() *render.FontAtlas { return a.text.Atlas }

// Clipboard returns the platform clipboard.
func (a *Window) Clipboard() input.Clipboard { return a.clip }

// buildAppFrame builds and solves one frame via the shared ui driver.
func (a *Window) buildAppFrame(app App, w, h float32) *layout.Element {
	return ui.BuildFrame(a.uiCtx, app.Body, w, h, a.mouse, a.keyboard)
}

// paintAppFrame rebuilds the body and submits a GPU frame.
func (a *Window) paintAppFrame(app App, w, h float32) {
	a.presentFrame(a.buildAppFrame(app, w, h))
}

// presentFrame paints an already-built root and submits a GPU frame.
func (a *Window) presentFrame(root *layout.Element) {
	a.renderer.ClearColor = theme.Current().Surface
	a.drawList.Reset()
	a.text.Atlas.BindDrawList(&a.drawList)
	layout.Paint(root, &a.drawList, a.text)
	a.text.Atlas.BindDrawList(nil)
	_ = a.text.FlushAtlas(a.renderer)
	if err := a.renderer.Render(&a.drawList); err != nil && !transientSurfaceError(err) {
		fmt.Println("render error:", err)
	}
}

// routeAppKeys lets the app consume key events before focus routing.
func (a *Window) routeAppKeys(hook KeyHook) {
	keys := a.keyboard.Keys
	kept := keys[:0]
	for _, k := range keys {
		if !hook.OnKey(a.uiCtx, k) {
			kept = append(kept, k)
		}
	}
	a.keyboard.Keys = kept
}

func transientSurfaceError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Surface timed out") ||
		strings.Contains(s, "Surface is outdated") ||
		strings.Contains(s, "Surface was lost")
}
