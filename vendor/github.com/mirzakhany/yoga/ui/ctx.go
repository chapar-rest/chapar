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
// not retain per-frame fields (mouse, overlays, theme snapshot) across frames.
// The Ctx pointer itself is window-lifetime, as are Dialogs/Files/Toasts and
// Focus. App state lives in retained app structs; widget micro-state (hover,
// caret, scroll) lives in the id-keyed store.
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
	dialogs  *DialogHost   // window-owned; laid out by BuildFrame
	files    *FileDialog
	toasts   *ToastHost
	commands *CommandsHost
	window   WindowHost
	mouse    *input.Mouse
	keyboard *input.Keyboard
	post     func() // thread-safe wake (glfw.PostEmptyEvent); may be nil
	store    *widgetStore
	env      env
	autoSeq  int
}

// New builds a Ctx bound to a text engine and an optional thread-safe wake.
func New(text *shape.Engine, focus *FocusScope, post func()) *Ctx {
	return &Ctx{
		text:     text,
		focus:    focus,
		post:     post,
		wakeIn:   -1,
		store:    newStore(),
		dialogs:  NewDialogHost(),
		files:    NewFileDialog(),
		toasts:   NewToastHost(),
		commands: NewCommandsHost(),
	}
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
	if c.commands != nil {
		c.commands.beginFrame()
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

// Dialogs returns the window-owned dialog host. Show from Body or OnClick;
// BuildFrame lays it out so it does not need to be in the view tree.
func (c *Ctx) Dialogs() *DialogHost {
	if c.dialogs == nil {
		c.dialogs = NewDialogHost()
	}
	return c.dialogs
}

// Files returns the window-owned file picker. Show from Body or OnClick;
// BuildFrame lays it out so it does not need to be in the view tree.
func (c *Ctx) Files() *FileDialog {
	if c.files == nil {
		c.files = NewFileDialog()
	}
	return c.files
}

// Toasts returns the window-owned toast stack. Show from Body or OnClick;
// BuildFrame lays it out so it does not need to be in the view tree.
func (c *Ctx) Toasts() *ToastHost {
	if c.toasts == nil {
		c.toasts = NewToastHost()
	}
	return c.toasts
}

// Commands returns the window-owned command registry and palette.
// Register commands from Body; Show/Toggle from Body or OnClick; BuildFrame
// lays the palette out so it does not need to be in the view tree.
func (c *Ctx) Commands() *CommandsHost {
	if c.commands == nil {
		c.commands = NewCommandsHost()
	}
	return c.commands
}

// layoutWindowOverlays registers the window-owned overlay hosts after the
// body so dialogs stack above the page, the file picker above dialogs,
// the command palette above those, and toasts on top. Must run before
// FocusScope.finishBuild so modal Add/SetModal participate in this frame's
// focus list.
func (c *Ctx) layoutWindowOverlays() {
	if c.dialogs != nil {
		c.dialogs.Layout(c)
	}
	if c.files != nil {
		c.files.Layout(c)
	}
	if c.commands != nil {
		c.commands.Layout(c)
	}
	if c.toasts != nil {
		c.toasts.Layout(c)
	}
}

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
