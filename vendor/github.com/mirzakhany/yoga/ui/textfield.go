package ui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const textFieldBlink = 500 * time.Millisecond

// TextFieldConfig configures a single-line text input.
type TextFieldConfig struct {
	Placeholder string
	Password    bool
	IconStart   icons.Icon
	IconEnd     icons.Icon
	Radius      float32
	BorderWidth float32
	Height      float32
}

// TextField is a single-line text input with optional border, rounded corners,
// start/end icons, and password masking.
type TextInput struct {
	host *layout.Element
	cfg  TextFieldConfig

	Value    string
	OnChange func(value string)
	OnSubmit func(value string) // fired when Enter is pressed while focused

	focused    bool
	caret      int // byte offset
	selAnchor  int // byte offset where a selection began, or -1 for none
	dragging   bool
	caretShown bool
	blinkStart time.Time
	// scrollX shifts the text left so the caret stays inside the viewport when
	// the content is wider than the field.
	scrollX float32

	// Multi-click tracking for word (double) and select-all (triple) selection.
	lastClickTime time.Time
	lastClickX    float32
	clickCount    int
}

// NewTextField builds a text field with the given configuration.
func NewTextInput(cfg TextFieldConfig) *TextInput {
	th := theme.Current()
	if cfg.Radius <= 0 {
		cfg.Radius = th.Radius.Medium
	}
	if cfg.BorderWidth <= 0 {
		cfg.BorderWidth = th.Stroke.Thin
	}
	if cfg.Height <= 0 {
		cfg.Height = th.Metrics.ControlHeight
	}
	tf := &TextInput{
		cfg:        cfg,
		blinkStart: time.Now(),
		caretShown: true,
		selAnchor:  -1,
	}
	padX := th.Spacing.MNudge
	st := layout.Box().H(cfg.Height).FlexShrink(0).PaddingXY(padX, 0)
	st.MinHeight = cfg.Height
	tf.host = layout.New(st)
	tf.host.Paint = tf.paint
	tf.host.OnMouse = tf.onMouse
	return tf
}

// minWidth is padding + icons + a short text slot. Icons and placeholder are
// painted, not layout children, so the engine's intrinsic width is otherwise
// only horizontal padding — a Row + Spacer would crush the field.
func (tf *TextInput) minWidth() float32 {
	th := theme.Current()
	w := 2 * tf.padX()
	if !tf.cfg.IconStart.Empty() {
		w += tf.iconSize() + tf.iconGap()
	}
	if !tf.cfg.IconEnd.Empty() {
		w += tf.iconSize() + tf.iconGap()
	}
	slot := "MMMMMMMM"
	if text := frameText(); text != nil {
		tw, _ := text.MeasureAt(slot, th.Typography.Body.Size)
		w += tw
	} else {
		w += float32(len(slot)) * th.Typography.Body.Size
	}
	return w
}

// Focus grants keyboard focus to the field.
func (tf *TextInput) Focus() {
	tf.focused = true
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

// Focused reports whether the field has keyboard focus.
func (tf *TextInput) Focused() bool { return tf.focused }

// Blur removes keyboard focus.
func (tf *TextInput) Blur() { tf.focused = false }

// CapturesTab reports that plain Tab should move focus rather than insert text.
func (tf *TextInput) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the field should grant focus.
func (tf *TextInput) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (tf *TextInput) FocusEl() *layout.Element { return tf.host }

func (tf *TextInput) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(tf)
	tf.Update(c.Mouse())
	if tf.focused {
		since := time.Since(tf.blinkStart) % (2 * textFieldBlink)
		wait := textFieldBlink - (since % textFieldBlink)
		c.Animate(wait)
	}
	return tf.host
}

func (tf *TextInput) displayText() string {
	if tf.cfg.Password {
		return strings.Repeat("•", utf8.RuneCountInString(tf.Value))
	}
	return tf.Value
}

func (tf *TextInput) padX() float32 { return theme.Current().Spacing.MNudge }

func (tf *TextInput) iconSize() float32 { return theme.Current().Metrics.IconSizeSM }

func (tf *TextInput) iconGap() float32 { return theme.Current().Spacing.SNudge }

func (tf *TextInput) textLeft() float32 {
	x := tf.host.Frame.X + tf.padX()
	if !tf.cfg.IconStart.Empty() {
		x += tf.iconSize() + tf.iconGap()
	}
	return x
}

func (tf *TextInput) textRight() float32 {
	x := tf.host.Frame.X + tf.host.Frame.W - tf.padX()
	if !tf.cfg.IconEnd.Empty() {
		x -= tf.iconSize() + tf.iconGap()
	}
	return x
}

// displayPrefixFor returns the display text up to the given Value byte offset,
// accounting for password masking (which maps runes 1:1 but changes byte widths).
func (tf *TextInput) displayPrefixFor(valueOff int) string {
	disp := tf.displayText()
	runes := 0
	for i := 0; i < len(tf.Value) && i < valueOff; {
		_, sz := utf8.DecodeRuneInString(tf.Value[i:])
		i += sz
		runes++
	}
	ri := 0
	for count := 0; count < runes && ri < len(disp); count++ {
		_, sz := utf8.DecodeRuneInString(disp[ri:])
		ri += sz
	}
	return disp[:ri]
}

// offsetXForValue returns the x position of a Value byte offset relative to the
// start of the text (before applying scrollX).
func (tf *TextInput) offsetXForValue(valueOff int) float32 {
	style := theme.Current().Typography.Body
	tw, _ := frameText().MeasureAt(tf.displayPrefixFor(valueOff), style.Size)
	return tw
}

// caretOffsetX is the caret's x position relative to the start of the text
// (before applying scrollX).
func (tf *TextInput) caretOffsetX() float32 {
	return tf.offsetXForValue(tf.caret)
}

func (tf *TextInput) caretX() float32 {
	return tf.textLeft() - tf.scrollX + tf.caretOffsetX()
}

// ensureCaretVisible adjusts scrollX so the caret sits inside the text
// viewport. Call when the frame is valid (paint time).
func (tf *TextInput) ensureCaretVisible() {
	th := theme.Current()
	style := th.Typography.Body
	viewW := tf.textRight() - tf.textLeft()
	if viewW <= 0 {
		tf.scrollX = 0
		return
	}
	fullW, _ := frameText().MeasureAt(tf.displayText(), style.Size)
	caretPad := th.Stroke.Thick
	maxScroll := f32max(0, fullW+caretPad-viewW)
	tf.scrollX = clampf(tf.scrollX, 0, maxScroll)

	cx := tf.caretOffsetX()
	if cx-tf.scrollX < 0 {
		tf.scrollX = cx
	} else if cx-tf.scrollX > viewW-caretPad {
		tf.scrollX = cx - viewW + caretPad
	}
	tf.scrollX = clampf(tf.scrollX, 0, maxScroll)
}

// offsetAtX maps a pixel x to the nearest Value byte offset.
func (tf *TextInput) offsetAtX(px float32) int {
	s := tf.displayText()
	style := theme.Current().Typography.Body
	x0 := tf.textLeft() - tf.scrollX
	// Shape at the same logical size used for painting so click-to-caret
	// matches glyph positions when the theme body size is not the default.
	ln := frameText().LineAt(s, style.Size)
	off := ln.ByteForX(px - x0)
	if tf.cfg.Password {
		// Map display byte offset back to Value byte offset by rune count.
		runes := 0
		for i := 0; i < off && i < len(s); {
			_, sz := utf8.DecodeRuneInString(s[i:])
			i += sz
			runes++
		}
		off = 0
		for count := 0; count < runes && off < len(tf.Value); count++ {
			_, sz := utf8.DecodeRuneInString(tf.Value[off:])
			off += sz
		}
	}
	return off
}

func (tf *TextInput) selRange() (int, int) {
	if tf.selAnchor < 0 || tf.selAnchor == tf.caret {
		return tf.caret, tf.caret
	}
	if tf.selAnchor < tf.caret {
		return tf.selAnchor, tf.caret
	}
	return tf.caret, tf.selAnchor
}

func (tf *TextInput) hasSelection() bool {
	return tf.selAnchor >= 0 && tf.selAnchor != tf.caret
}

// deleteSelection removes the selected text, collapsing the caret to its start.
// Returns true if a selection was present.
func (tf *TextInput) deleteSelection() bool {
	if !tf.hasSelection() {
		return false
	}
	lo, hi := tf.selRange()
	tf.caret = lo
	tf.selAnchor = -1
	tf.setValue(tf.Value[:lo] + tf.Value[hi:])
	return true
}

// moveTo places the caret at off, extending the selection when extend is true
// and collapsing it otherwise.
func (tf *TextInput) moveTo(off int, extend bool) {
	if off < 0 {
		off = 0
	}
	if off > len(tf.Value) {
		off = len(tf.Value)
	}
	if extend {
		if tf.selAnchor < 0 {
			tf.selAnchor = tf.caret
		}
	} else {
		tf.selAnchor = -1
	}
	tf.caret = off
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

func (tf *TextInput) setValue(s string) {
	// Single line: strip newlines.
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	if s == tf.Value {
		tf.clampCaret()
		return
	}
	tf.Value = s
	tf.clampCaret()
	if tf.OnChange != nil {
		tf.OnChange(s)
	}
}

func (tf *TextInput) clampCaret() {
	if tf.caret < 0 {
		tf.caret = 0
	}
	if tf.caret > len(tf.Value) {
		tf.caret = len(tf.Value)
	}
}

func (tf *TextInput) insertAtCaret(s string) {
	if s == "" {
		return
	}
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return
	}
	tf.deleteSelection()
	tf.clampCaret()
	tf.setValue(tf.Value[:tf.caret] + s + tf.Value[tf.caret:])
	tf.caret += len(s)
	tf.selAnchor = -1 // collapse click/drag anchor so the insert is not selected
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

func (tf *TextInput) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	text := frameText()
	sheet := frameIcons()
	f := tf.host.Frame
	border := th.Border
	if tf.focused {
		border = th.FocusRing
		// Soft 2px accent glow around the field (box-shadow-style focus ring).
		glow := render.Rect{X: f.X - 2, Y: f.Y - 2, W: f.W + 4, H: f.H + 4}
		dl.AddRoundedRect(glow, tf.cfg.Radius+2, render.Color{R: th.FocusRing.R, G: th.FocusRing.G, B: th.FocusRing.B, A: 0.25})
	}
	dl.AddRoundedRectBorder(f, tf.cfg.Radius, tf.cfg.BorderWidth, th.Chrome, border)

	iconSz := tf.iconSize()
	iconY := f.Y + (f.H-iconSz)/2
	if !tf.cfg.IconStart.Empty() {
		ix := f.X + tf.padX()
		sheet.Draw(dl, tf.cfg.IconStart, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, th.ForegroundMuted)
	}
	if !tf.cfg.IconEnd.Empty() {
		ix := f.X + f.W - tf.padX() - iconSz
		sheet.Draw(dl, tf.cfg.IconEnd, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, th.ForegroundMuted)
	}

	tx := tf.textLeft()
	tr := tf.textRight()
	style := th.Typography.Body
	_, lh := text.MeasureAt("Ag", style.Size)
	ty := f.Y + (f.H-lh)/2

	// Keep the caret inside the viewport by scrolling the text horizontally.
	tf.ensureCaretVisible()

	show := tf.displayText()
	col := th.Foreground
	if show == "" && !tf.focused {
		show = tf.cfg.Placeholder
		col = th.ForegroundMuted
	}
	if show != "" || tf.hasSelection() {
		dl.PushClip(render.Rect{X: tx, Y: f.Y, W: tr - tx, H: f.H})
		// Selection highlight, drawn behind the glyphs.
		if tf.focused {
			if selLo, selHi := tf.selRange(); selLo != selHi {
				xLo := clampf(tf.textLeft()-tf.scrollX+tf.offsetXForValue(selLo), tx, tr)
				xHi := clampf(tf.textLeft()-tf.scrollX+tf.offsetXForValue(selHi), tx, tr)
				if xHi > xLo {
					dl.AddRect(render.Rect{X: xLo, Y: ty, W: xHi - xLo, H: lh}, th.Selection)
				}
			}
		}
		text.DrawStringTopAt(dl, show, tx-tf.scrollX, ty, col, style.Size)
		dl.PopClip()
	}

	if tf.focused && tf.caretShown {
		cx := clampf(tf.caretX(), tx, tr)
		dl.AddRect(render.Rect{X: cx, Y: ty, W: th.Stroke.Thick, H: lh}, th.Accent)
	}
}

func (tf *TextInput) onMouse(e *layout.Element, m *input.Mouse) {
	if m.Pressed && e.Frame.Contains(m.X, m.Y) {
		off := tf.offsetAtX(m.X)
		now := time.Now()
		if now.Sub(tf.lastClickTime) < doubleClickInterval && absF(m.X-tf.lastClickX) <= multiClickSlop {
			tf.clickCount++
			if tf.clickCount > 3 {
				tf.clickCount = 1
			}
		} else {
			tf.clickCount = 1
		}
		tf.lastClickTime = now
		tf.lastClickX = m.X

		switch tf.clickCount {
		case 2: // word selection
			lo, hi := wordRangeIn(tf.Value, off)
			tf.selAnchor = lo
			tf.caret = hi
			tf.dragging = false
		case 3: // select all
			tf.selAnchor = 0
			tf.caret = len(tf.Value)
			tf.dragging = false
		default: // single click: place caret and start a drag selection
			tf.caret = off
			tf.selAnchor = off
			tf.dragging = true
		}
		tf.blinkStart = time.Now()
		tf.caretShown = true
		m.Consumed = true
	}
	// Drag extends the selection, even past the field's edges.
	if tf.dragging && m.Down {
		tf.caret = tf.offsetAtX(m.X)
	}
	if !m.Down {
		tf.dragging = false
		// Click without a drag is a caret placement, not a pending selection.
		if tf.selAnchor == tf.caret {
			tf.selAnchor = -1
		}
	}
}

// wordRangeIn returns the byte range of the "word" surrounding off in s: a run
// of like-classed runes (identifier characters, whitespace, or punctuation).
// When off sits just past a word it expands the word to the left.
func wordRangeIn(s string, off int) (int, int) {
	n := len(s)
	if n == 0 {
		return 0, 0
	}
	if off > n {
		off = n
	}
	cls := classSpace
	if off < n {
		r, _ := utf8.DecodeRuneInString(s[off:])
		cls = classOf(r)
	}
	if cls != classWord && off > 0 {
		pr, _ := utf8.DecodeLastRuneInString(s[:off])
		if classOf(pr) == classWord {
			off = prevRuneOff(s, off)
			cls = classWord
		}
	}
	if off >= n {
		off = prevRuneOff(s, n)
	}
	lo := off
	for lo > 0 {
		pr, sz := utf8.DecodeLastRuneInString(s[:lo])
		if classOf(pr) != cls {
			break
		}
		lo -= sz
	}
	hi := off
	for hi < n {
		r, sz := utf8.DecodeRuneInString(s[hi:])
		if classOf(r) != cls {
			break
		}
		hi += sz
	}
	return lo, hi
}

// Update advances caret blink; call once per frame.
func (tf *TextInput) Update(_ *input.Mouse) {
	if !tf.focused {
		return
	}
	if time.Since(tf.blinkStart) >= textFieldBlink {
		tf.caretShown = !tf.caretShown
		tf.blinkStart = time.Now()
	}
}

// HandleText inserts text-producing characters for this frame.
func (tf *TextInput) HandleText(runes []rune) {
	if !tf.focused || len(runes) == 0 {
		return
	}
	tf.insertAtCaret(string(runes))
}

// ── Builder/modifier methods ─────────────────────────────────────────────────

// Changed sets the OnChange callback.
func (tf *TextInput) Changed(fn func(string)) *TextInput { tf.OnChange = fn; return tf }

// WithIconStart sets the leading icon.
func (tf *TextInput) WithIconStart(icon icons.Icon) *TextInput { tf.cfg.IconStart = icon; return tf }

// WithIconEnd sets the trailing icon.
func (tf *TextInput) WithIconEnd(icon icons.Icon) *TextInput { tf.cfg.IconEnd = icon; return tf }

// AsPassword enables password masking (displays bullets instead of characters).
func (tf *TextInput) AsPassword() *TextInput { tf.cfg.Password = true; return tf }

// HandleKeys processes navigation and editing keys for this frame.
func (tf *TextInput) HandleKeys(keys []input.KeyEvent) {
	if !tf.focused {
		return
	}
	clip := frameClipboard()
	for _, ev := range keys {
		shift := ev.Mods.Has(input.ModShift)
		if ev.Key == input.KeyEnter && !ev.Mods.Primary() {
			if tf.OnSubmit != nil {
				tf.OnSubmit(tf.Value)
			}
			continue
		}
		if ev.Mods.Primary() {
			switch ev.Key {
			case input.KeyA:
				tf.selAnchor = 0
				tf.caret = len(tf.Value)
			case input.KeyC:
				if clip != nil {
					if lo, hi := tf.selRange(); lo != hi {
						clip.Set(tf.Value[lo:hi])
					} else if tf.Value != "" {
						clip.Set(tf.Value)
					}
				}
			case input.KeyX:
				if clip != nil {
					if tf.hasSelection() {
						lo, hi := tf.selRange()
						clip.Set(tf.Value[lo:hi])
						tf.deleteSelection()
					} else if tf.Value != "" {
						clip.Set(tf.Value)
						tf.selAnchor = -1
						tf.caret = 0
						tf.setValue("")
					}
				}
			case input.KeyV:
				if clip != nil {
					tf.insertAtCaret(clip.Get())
				}
			}
			continue
		}
		switch ev.Key {
		case input.KeyBackspace:
			if tf.deleteSelection() {
				break
			}
			if tf.caret > 0 {
				prev := prevRuneOff(tf.Value, tf.caret)
				tf.setValue(tf.Value[:prev] + tf.Value[tf.caret:])
				tf.caret = prev
			}
		case input.KeyDelete:
			if tf.deleteSelection() {
				break
			}
			if tf.caret < len(tf.Value) {
				next := nextRuneOff(tf.Value, tf.caret)
				tf.setValue(tf.Value[:tf.caret] + tf.Value[next:])
			}
		case input.KeyLeft:
			if tf.hasSelection() && !shift {
				lo, _ := tf.selRange()
				tf.moveTo(lo, false)
			} else {
				tf.moveTo(prevRuneOff(tf.Value, tf.caret), shift)
			}
		case input.KeyRight:
			if tf.hasSelection() && !shift {
				_, hi := tf.selRange()
				tf.moveTo(hi, false)
			} else {
				tf.moveTo(nextRuneOff(tf.Value, tf.caret), shift)
			}
		case input.KeyHome:
			tf.moveTo(0, shift)
		case input.KeyEnd:
			tf.moveTo(len(tf.Value), shift)
		}
	}
}

func prevRuneOff(s string, off int) int {
	if off <= 0 {
		return 0
	}
	_, sz := utf8.DecodeLastRuneInString(s[:off])
	return off - sz
}

func nextRuneOff(s string, off int) int {
	if off >= len(s) {
		return len(s)
	}
	_, sz := utf8.DecodeRuneInString(s[off:])
	return off + sz
}
