package ui

import (
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Ctx is the per-frame build context (the "gtx"). A single value is threaded
// through Body and every View.Layout call. It is the one place apps and
// widgets reach for invalidation, animation scheduling, overlay registration,
// the widget store, and frame-wide state (viewport, theme, text engine).
//
// A Ctx is owned by the runtime and reset at the start of each build pass; do
// not retain it across frames. App state lives in retained app structs; widget
// micro-state (hover, caret, scroll) lives in the id-keyed store.
type Ctx struct {
	now      time.Time
	vw, vh   float32
	text     *shape.Engine
	icons    *render.SpriteSheet
	clip     input.Clipboard
	theme    *theme.Theme
	overlays []*layout.Element
	wakeIn   time.Duration // smallest Animate() request this frame; <0 = none
	focus    *FocusScope   // runtime-owned, persists across frames
	mouse    *input.Mouse
	keyboard *input.Keyboard
	post     func() // thread-safe wake (glfw.PostEmptyEvent); may be nil
	store    *widgetStore
	env      env
	autoSeq  int
}

// New builds a Ctx bound to a text engine and an optional thread-safe wake.
func New(text *shape.Engine, focus *FocusScope, post func()) *Ctx {
	return &Ctx{text: text, focus: focus, post: post, wakeIn: -1, store: newStore()}
}

// BeginFrame resets per-frame state and records the frame clock, viewport, and
// this frame's input. The runtime calls this immediately before building the
// body.
func (c *Ctx) BeginFrame(vw, vh float32, m *input.Mouse, kb *input.Keyboard) {
	c.now = time.Now()
	c.vw, c.vh = vw, vh
	c.mouse, c.keyboard = m, kb
	c.theme = theme.Current()
	c.overlays = c.overlays[:0]
	c.wakeIn = -1
	c.env = env{}
	c.autoSeq = 0
	c.beginStorePass()
	c.bindFrameResources()
	if c.focus != nil {
		c.focus.beginFrame()
	}
	SetViewport(vw, vh)
}

// Mouse returns this frame's pointer state. May be nil in tests.
func (c *Ctx) Mouse() *input.Mouse { return c.mouse }

// Keyboard returns this frame's keyboard state. May be nil in tests.
func (c *Ctx) Keyboard() *input.Keyboard { return c.keyboard }

// Invalidate requests a repaint. Safe to call from any goroutine.
func (c *Ctx) Invalidate() {
	if c.post != nil {
		c.post()
	}
}

// Focus returns the runtime-owned focus scope.
func (c *Ctx) Focus() *FocusScope { return c.focus }

// Animate requests a repaint within d. The runtime takes the minimum across
// all Animate calls this frame as its wait budget.
func (c *Ctx) Animate(d time.Duration) {
	if d < 0 {
		d = 0
	}
	if c.wakeIn < 0 || d < c.wakeIn {
		c.wakeIn = d
	}
}

// AnimationWait reports the aggregated animation budget for this frame.
func (c *Ctx) AnimationWait() (time.Duration, bool) {
	if c.wakeIn < 0 {
		return 0, false
	}
	return c.wakeIn, true
}

// Overlay registers portal content for this frame.
func (c *Ctx) Overlay(e *layout.Element) {
	if e != nil {
		c.overlays = append(c.overlays, e)
	}
}

// Overlays returns the overlay elements registered this frame.
func (c *Ctx) Overlays() []*layout.Element { return c.overlays }

// Now is the single frame clock.
func (c *Ctx) Now() time.Time {
	if c.now.IsZero() {
		return time.Now()
	}
	return c.now
}

// Viewport returns the current drawable size in logical pixels.
func (c *Ctx) Viewport() (w, h float32) { return c.vw, c.vh }

// Theme returns the active theme for this frame.
func (c *Ctx) Theme() *theme.Theme {
	if c.theme == nil {
		return theme.Current()
	}
	return c.theme
}

// Text returns the shared shaping/measuring engine.
func (c *Ctx) Text() *shape.Engine { return c.text }
