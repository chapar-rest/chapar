// Package input holds the platform-agnostic pointer state for a frame. It has
// no dependency on GLFW or on the layout package, which keeps it a pure-Go leaf
// that other packages (layout, components) can import without creating an import
// cycle. The example translates raw GLFW callbacks into a Mouse via the setter
// methods below.
package input

// Cursor is the OS pointer shape a widget requests for the current frame.
type Cursor int

const (
	CursorDefault Cursor = iota
	CursorPointer
	CursorResizeEW
	CursorResizeNS
)

// Mouse is the per-frame pointer state. The example mutates it from GLFW
// callbacks; components read it during their Update phase.
//
// Edge flags (Pressed/Released) are valid for exactly one frame and must be
// cleared by calling EndFrame() after event dispatch.
type Mouse struct {
	X, Y float32 // current cursor position in pixels (top-left origin)

	Down     bool // true while the primary button is held
	Pressed  bool // true on the frame the primary button went down
	Released bool // true on the frame the primary button went up

	// Secondary (right) button edges, used for context menus.
	RightDown     bool
	RightPressed  bool
	RightReleased bool

	ScrollX float32 // accumulated horizontal wheel delta for this frame
	ScrollY float32 // accumulated vertical wheel delta for this frame

	// Consumed lets a front (overlay) widget claim the event so widgets behind
	// it ignore the same click during dispatch.
	Consumed bool

	// Cursor is the pointer shape widgets request for this frame. The runtime
	// resets it to CursorDefault before Update; components set it during dispatch.
	Cursor Cursor

	// Mods is the modifier set at the last mouse-button event (click to select
	// with Shift / Cmd). It is not cleared by EndFrame so it stays valid for
	// the whole click frame.
	Mods Mod

	prevDown      bool
	prevRightDown bool
}

// SetPos updates the cursor position.
func (m *Mouse) SetPos(x, y float32) {
	m.X, m.Y = x, y
}

// SetButton records the primary button state and derives the edge flags.
func (m *Mouse) SetButton(down bool) {
	if down && !m.prevDown {
		m.Pressed = true
	}
	if !down && m.prevDown {
		m.Released = true
	}
	m.Down = down
	m.prevDown = down
}

// SetRightButton records the secondary (right) button state and derives edges.
func (m *Mouse) SetRightButton(down bool) {
	if down && !m.prevRightDown {
		m.RightPressed = true
	}
	if !down && m.prevRightDown {
		m.RightReleased = true
	}
	m.RightDown = down
	m.prevRightDown = down
}

// AddScroll accumulates vertical wheel movement for the current frame.
func (m *Mouse) AddScroll(dy float32) {
	m.ScrollY += dy
}

// AddScrollX accumulates horizontal wheel movement for the current frame.
func (m *Mouse) AddScrollX(dx float32) {
	m.ScrollX += dx
}

// SetCursor requests an OS pointer shape for this frame.
func (m *Mouse) SetCursor(c Cursor) { m.Cursor = c }

// EndFrame clears the one-frame edge flags. Call once per frame after all
// widgets have processed the mouse.
func (m *Mouse) EndFrame() {
	m.Pressed = false
	m.Released = false
	m.RightPressed = false
	m.RightReleased = false
	m.ScrollX = 0
	m.ScrollY = 0
	m.Consumed = false
}

// Key is a platform-agnostic keyboard key identifier. Only the subset the
// editor needs is modeled; the example maps raw GLFW keys onto these.
type Key int

const (
	KeyNone Key = iota
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyBackspace
	KeyDelete
	KeyEnter
	KeyTab
	KeyA
	KeyC
	KeyV
	KeyX
	KeyZ
	KeyS
	KeyF
	KeyH
	KeyEscape
	KeySpace
)

// Mod is a bitset of active modifier keys.
type Mod uint8

const (
	ModShift Mod = 1 << iota
	ModCtrl
	ModAlt
	ModSuper // Command on macOS
)

// Has reports whether all bits in m are set.
func (mods Mod) Has(m Mod) bool { return mods&m == m }

// Primary reports whether the platform's primary shortcut modifier is held
// (Command on macOS, Control elsewhere). We accept either so the same shortcut
// works everywhere without per-OS branching at the call site.
func (mods Mod) Primary() bool { return mods.Has(ModSuper) || mods.Has(ModCtrl) }

// KeyEvent is a single non-text key press (or auto-repeat) with its modifiers.
type KeyEvent struct {
	Key  Key
	Mods Mod
}

// Keyboard accumulates text and key events for one frame. Like Mouse, it must be
// cleared with EndFrame after the focused widget has consumed it.
//
// Chars are produced by the OS text-input path (GLFW char callback), already
// accounting for layout/shift; Keys carries navigation/editing/shortcut keys
// that do not produce text.
type Keyboard struct {
	Chars []rune
	Keys  []KeyEvent
}

// TypeRune records a text-producing character for this frame.
func (k *Keyboard) TypeRune(r rune) { k.Chars = append(k.Chars, r) }

// PressKey records a non-text key event for this frame.
func (k *Keyboard) PressKey(key Key, mods Mod) {
	k.Keys = append(k.Keys, KeyEvent{Key: key, Mods: mods})
}

// EndFrame clears the per-frame buffers.
func (k *Keyboard) EndFrame() {
	k.Chars = k.Chars[:0]
	k.Keys = k.Keys[:0]
}

// Clipboard abstracts the system clipboard so the editor (in the components
// package) stays free of any windowing dependency. The example wires a
// GLFW-backed implementation; headless builds use a no-op or in-memory one.
type Clipboard interface {
	Get() string
	Set(string)
}

// MemClipboard is an in-process Clipboard, handy for headless/testing.
type MemClipboard struct{ data string }

func (c *MemClipboard) Get() string  { return c.data }
func (c *MemClipboard) Set(s string) { c.data = s }
