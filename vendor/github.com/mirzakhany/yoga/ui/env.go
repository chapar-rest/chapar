package ui

import (
	"fmt"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type env struct {
	textColor render.Color
	fontSize  float32
	hasColor  bool
	hasSize   bool
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
	return old
}

func (c *Ctx) popEnv(old env) { c.env = old }

func autoID(c *Ctx, prefix string) string {
	c.autoSeq++
	return fmt.Sprintf("%s-%d", prefix, c.autoSeq)
}

var (
	activeText  *shape.Engine
	activeIcons *render.SpriteSheet
	activeClip  input.Clipboard
)

func frameText() *shape.Engine        { return activeText }
func frameIcons() *render.SpriteSheet { return activeIcons }
func frameClipboard() input.Clipboard { return activeClip }

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
}

// Icons returns the sprite sheet bound to this context.
func (c *Ctx) Icons() *render.SpriteSheet { return c.icons }

// Clipboard returns the platform clipboard bound to this context.
func (c *Ctx) Clipboard() input.Clipboard { return c.clip }

// SetIcons attaches a sprite sheet used by Icon/Button paints.
func (c *Ctx) SetIcons(s *render.SpriteSheet) { c.icons = s }

// SetClipboard attaches a clipboard used by text fields and the editor.
func (c *Ctx) SetClipboard(clip input.Clipboard) { c.clip = clip }
