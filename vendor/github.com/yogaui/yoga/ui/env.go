package ui

import (
	"fmt"

	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/shape"
)

type env struct {
	textColor        render.Color
	fontSize         float32
	hasColor         bool
	hasSize          bool
	controlHeight    float32
	hasControlHeight bool
	joined           joined // set while an InputGroup lays out its parts
}

func (c *Ctx) pushEnv(e env) env {
	old := c.env
	if e.hasColor {
		c.env.textColor = e.textColor
		c.env.hasColor = true
	}
	if e.hasSize {
		c.env.fontSize = e.fontSize
		c.env.hasSize = true
	}
	if e.hasControlHeight {
		c.env.controlHeight = e.controlHeight
		c.env.hasControlHeight = true
	}
	return old
}

func (c *Ctx) popEnv(old env) { c.env = old }

// controlHeight returns the active compact control height. TitleBar pushes a
// smaller TitleBarControlHeight so widgets fit with vertical breathing room.
func (c *Ctx) controlHeight() float32 {
	if c.env.hasControlHeight {
		return c.env.controlHeight
	}
	return c.Theme().Metrics.ControlHeight
}

func autoID(c *Ctx, prefix string) string {
	c.autoSeq++
	return fmt.Sprintf("%s-%d", prefix, c.autoSeq)
}

var (
	activeText  *shape.Engine
	activeIcons *render.SpriteSheet
	activeClip  input.Clipboard
)

func FrameText() *shape.Engine        { return activeText }
func FrameIcons() *render.SpriteSheet { return activeIcons }
func FrameClipboard() input.Clipboard { return activeClip }

// SetFrameResources binds the text engine, icon sheet, and clipboard used by
// widgets that paint outside Layout (tests and headless drivers).
func SetFrameResources(text *shape.Engine, icons *render.SpriteSheet, clip input.Clipboard) {
	activeText = text
	activeIcons = icons
	activeClip = clip
}

func (c *Ctx) bindFrameResources() {
	activeText = c.text
	activeIcons = c.icons
	activeClip = c.clip
	activeFocus = c.focus
}

// activeFocus is the frame's focus scope, read by paint code to decide
// whether focus rings show.
var activeFocus *FocusScope

// focusVisible reports whether a focused widget draws its focus ring: only
// after keyboard interaction (see FocusScope.FocusVisible).
func focusVisible(focused bool) bool {
	return focused && (activeFocus == nil || activeFocus.FocusVisible())
}

// Icons returns the sprite sheet bound to this context.
func (c *Ctx) Icons() *render.SpriteSheet { return c.icons }

// Clipboard returns the platform clipboard bound to this context.
func (c *Ctx) Clipboard() input.Clipboard { return c.clip }

// SetIcons attaches a sprite sheet used by Icon/Button paints.
func (c *Ctx) SetIcons(s *render.SpriteSheet) { c.icons = s }

// SetClipboard attaches a clipboard used by text fields and the editor.
func (c *Ctx) SetClipboard(clip input.Clipboard) { c.clip = clip }

// Window returns the platform window host for custom title bars. Nil in headless builds.
func (c *Ctx) Window() WindowHost { return c.window }

// SetWindow attaches the platform window host used by TitleBar and WindowControls.
func (c *Ctx) SetWindow(w WindowHost) { c.window = w }

// WindowTitler is a WindowHost that can rename its window.
type WindowTitler interface {
	SetTitle(title string)
}

// SetWindowTitle sets the OS window title, which the Dock, Mission Control,
// the taskbar and window switchers show, even when a custom title bar hides
// it. It is cheap to call every frame; without a window it does nothing.
func (c *Ctx) SetWindowTitle(title string) {
	if t, ok := c.window.(WindowTitler); ok {
		t.SetTitle(title)
	}
}
