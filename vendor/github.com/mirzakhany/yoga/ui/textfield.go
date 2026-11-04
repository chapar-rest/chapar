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
	// CopyMasked lets a Password field copy and cut its real value: for
	// stored secrets the user owns, rather than a password being typed.
	CopyMasked  bool
	IconStart   icons.Icon
	IconEnd     icons.Icon
	Radius      float32
	BorderWidth float32
	Height      float32
	// Multiline makes the input a text area (see TextArea): newlines are
	// kept, long lines wrap, and the height follows the line count between
	// MinRows and MaxRows.
	Multiline        bool
	MinRows, MaxRows int
}

// TextField is a single-line text input with optional border, rounded corners,
// start/end icons, and password masking.
type TextInput struct {
	host   *layout.Element
	cfg    TextFieldConfig
	joined joined // place in an InputGroup, if any

	Value    string
	OnChange func(value string)
	OnSubmit func(value string) // fired when Enter is pressed while focused
	// ContextMenu, when set, edits the right-click menu: it receives the
	// standard edit items and returns the ones to show. Returning none turns
	// the menu off and leaves the right-click to widgets behind the field.
	ContextMenu func(items []MenuItem) []MenuItem
	// Highlight, when set, colors ranges of the value, such as template
	// placeholders. It runs once per paint.
	Highlight func(value string) []TextSpan
	// Suggest, when set, feeds the completion popup that opens while typing.
	Suggest SuggestFunc
	// HoverInfo, when set, explains the part of the value under a resting
	// pointer — what a template placeholder stands for, say.
	HoverInfo HoverInfoFunc

	focused    bool
	caret      int // byte offset
	selAnchor  int // byte offset where a selection began, or -1 for none
	dragging   bool
	caretShown bool
	blinkStart time.Time
	// scrollX shifts the text left so the caret stays inside the viewport when
	// the content is wider than the field.
	scrollX float32

	// Multi-line state: the vertical scroll, the x Up/Down keeps to (<0 for
	// none), the wrap of the value and the width it was made at, and the
	// caret and value followCaret last scrolled to.
	scrollY   float32
	goalX     float32
	spans     []textSpan
	wrapW     float32
	wrapText  string
	seenCaret int
	seenValue string

	// Multi-click tracking for word (double) and select-all (triple) selection.
	lastClickTime time.Time
	lastClickX    float32
	clickCount    int

	disabled   bool
	visualSpec Spec

	// Undo/redo keeps whole-value snapshots; a single line is small.
	undo       []textSnap
	redo       []textSnap
	lastMerge  editMerge // kind of the last edit, mergeNone after anything else
	lastEditAt time.Time

	menu  editMenu
	sugg  suggestState
	hover hoverState

	embed fieldEmbed // set when another widget hosts and draws the field
}

// fieldEmbed is how a widget that hosts a TextInput inside its own painting,
// such as the editor's find bar, styles it: the host draws the box, and padR
// keeps room on the right for the host's own buttons.
type fieldEmbed struct {
	on       bool
	padR     float32
	bg, line render.Color
	fg       render.Color
	hasFg    bool
}

// textSnap is a TextInput state that undo/redo returns to.
type textSnap struct {
	value         string
	caret, anchor int
}

// textInputUndoSteps bounds a TextInput's undo history.
const textInputUndoSteps = 100

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
		goalX:      -1,
		hover:      hoverState{off: -1},
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

// Blur removes keyboard focus and closes the right-click menu.
func (tf *TextInput) Blur() {
	tf.focused = false
	tf.menu.close()
	tf.closeSuggest()
}

// CapturesTab reports that plain Tab should move focus rather than insert text.
func (tf *TextInput) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the field should grant focus.
func (tf *TextInput) FocusOnClick() bool { return !tf.disabled }

// FocusEl returns the element used for click-to-focus hit testing.
func (tf *TextInput) FocusEl() *layout.Element { return tf.host }

func (tf *TextInput) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(tf)
	tf.Update(c.Mouse())
	tf.menu.layout(c)
	tf.layoutHoverCard(c)
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
	return x - tf.embed.padR
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
	tf.goalX = -1
	tf.lastMerge = mergeNone
	tf.blinkStart = time.Now()
	tf.caretShown = true
	// The suggestions described the old caret position.
	tf.closeSuggest()
}

// edit runs fn, which changes the value, and records the state before it as
// an undo step unless it continues the previous run of the same kind of edit.
// typed is the text a mergeType edit inserts.
func (tf *TextInput) edit(merge editMerge, typed string, fn func()) {
	before := textSnap{tf.Value, tf.caret, tf.selAnchor}
	tf.goalX = -1
	fn()
	if tf.Value == before.value {
		return
	}
	now := time.Now()
	// Typing over a selection or starting a new word opens a step, as does
	// any other kind of edit or a pause.
	newStep := merge == mergeNone || merge != tf.lastMerge || len(tf.undo) == 0 ||
		now.Sub(tf.lastEditAt) > undoMergeGap
	if merge == mergeType && !newStep {
		newStep = before.anchor >= 0 && before.anchor != before.caret ||
			startsWord(before.value[:before.caret], typed)
	}
	if newStep {
		tf.pushUndo(before)
	}
	tf.lastMerge = merge
	tf.lastEditAt = now
}

func (tf *TextInput) pushUndo(s textSnap) {
	if len(tf.undo) >= textInputUndoSteps {
		n := copy(tf.undo, tf.undo[1:])
		tf.undo = tf.undo[:n]
	}
	tf.undo = append(tf.undo, s)
	clear(tf.redo)
	tf.redo = tf.redo[:0]
}

// load replaces the value without recording it, and forgets the undo history,
// which described edits to a different value.
func (tf *TextInput) load(s string) {
	tf.setValue(s)
	tf.resetHistory()
}

func (tf *TextInput) resetHistory() {
	clear(tf.undo)
	clear(tf.redo)
	tf.undo, tf.redo = tf.undo[:0], tf.redo[:0]
	tf.lastMerge = mergeNone
}

// restore returns to a snapshot and gives back the state it replaced.
func (tf *TextInput) restore(s textSnap) textSnap {
	cur := textSnap{tf.Value, tf.caret, tf.selAnchor}
	tf.setValue(s.value)
	tf.caret, tf.selAnchor = s.caret, s.anchor
	tf.clampCaret()
	if tf.selAnchor > len(tf.Value) {
		tf.selAnchor = -1
	}
	tf.lastMerge = mergeNone
	tf.blinkStart = time.Now()
	tf.caretShown = true
	return cur
}

// CanUndo reports whether Undo has an edit to revert.
func (tf *TextInput) CanUndo() bool { return !tf.disabled && len(tf.undo) > 0 }

// CanRedo reports whether Redo has an undone edit to re-apply.
func (tf *TextInput) CanRedo() bool { return !tf.disabled && len(tf.redo) > 0 }

// Undo reverts the most recent edit.
func (tf *TextInput) Undo() {
	if !tf.CanUndo() {
		return
	}
	s := tf.undo[len(tf.undo)-1]
	tf.undo = tf.undo[:len(tf.undo)-1]
	tf.redo = append(tf.redo, tf.restore(s))
}

// Redo re-applies the most recently undone edit.
func (tf *TextInput) Redo() {
	if !tf.CanRedo() {
		return
	}
	s := tf.redo[len(tf.redo)-1]
	tf.redo = tf.redo[:len(tf.redo)-1]
	tf.undo = append(tf.undo, tf.restore(s))
}

func (tf *TextInput) setValue(s string) {
	s = tf.oneLine(s)
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

// oneLine keeps a single-line field's value to its first line; a multi-line
// field keeps every line, with line endings made LF.
func (tf *TextInput) oneLine(s string) string {
	if tf.cfg.Multiline {
		return normalizeNewlines(s)
	}
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	return s
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
	s = tf.oneLine(s)
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
	inter := interactStateFor(tf.disabled, false, false, tf.focused)
	r := tf.visualSpec.resolve(th, inter)
	bg := th.Chrome
	if r.hasBg {
		bg = r.bg
	}
	border := th.Border
	if tf.embed.on {
		border = tf.embed.line
		if tf.focused {
			border = th.FocusRing
		}
		dl.AddRoundedRectBorder(f, tf.cfg.Radius, tf.cfg.BorderWidth, tf.embed.bg, border)
	} else if tf.joined.in {
		paintJoinedFill(dl, f, tf.joined, bg, uniformRadius(r.radii, tf.cfg.Radius))
	} else if r.hasBorder {
		border = r.border
	} else if tf.focused && !tf.disabled {
		border = th.FocusRing
		// Soft 2px accent glow around the field (box-shadow-style focus ring).
		glow := render.Rect{X: f.X - 2, Y: f.Y - 2, W: f.W + 4, H: f.H + 4}
		dl.AddRoundedRect(glow, tf.cfg.Radius+2, render.Color{R: th.FocusRing.R, G: th.FocusRing.G, B: th.FocusRing.B, A: 0.25})
	}
	bw := uniformBorderWidth(r.borderW, tf.cfg.BorderWidth)
	radius := uniformRadius(r.radii, tf.cfg.Radius)
	if !tf.joined.in && !tf.embed.on {
		paintChromeBox(dl, f, r, bg, border, bw, radius)
	}

	iconSz := tf.iconSize()
	iconY := f.Y + (f.H-iconSz)/2
	if tf.cfg.Multiline {
		iconY = f.Y + (theme.Current().Metrics.ControlHeight-iconSz)/2
	}
	if !tf.cfg.IconStart.Empty() {
		ix := f.X + tf.padX()
		sheet.Draw(dl, tf.cfg.IconStart, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, th.ForegroundMuted)
	}
	if !tf.cfg.IconEnd.Empty() {
		ix := f.X + f.W - tf.padX() - iconSz
		sheet.Draw(dl, tf.cfg.IconEnd, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, th.ForegroundMuted)
	}

	if tf.cfg.Multiline {
		col := th.Foreground
		if tf.disabled {
			col = th.ForegroundDisabled
		} else if r.hasFg {
			col = r.fg
		}
		tf.paintLines(dl, col)
		return
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
	if tf.disabled {
		col = th.ForegroundDisabled
	} else if r.hasFg {
		col = r.fg
	}
	if tf.embed.hasFg {
		col = tf.embed.fg
	}
	if show == "" && !tf.focused {
		show = tf.cfg.Placeholder
		if !tf.disabled {
			col = th.ForegroundMuted
		}
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
		if spans := tf.highlightSpans(show); len(spans) > 0 {
			drawSpannedText(dl, text, show, spans, tx-tf.scrollX, ty, col, style.Size)
		} else {
			text.DrawStringTopAt(dl, show, tx-tf.scrollX, ty, col, style.Size)
		}
		dl.PopClip()
	}

	if tf.focused && tf.caretShown && !tf.disabled {
		cx := clampf(tf.caretX(), tx, tr)
		dl.AddRect(render.Rect{X: cx, Y: ty, W: th.Stroke.Thick, H: lh}, th.Accent)
	}
}

func (tf *TextInput) onMouse(e *layout.Element, m *input.Mouse) {
	if tf.disabled {
		return
	}
	tf.trackHover(e, m)
	if tf.cfg.Multiline && m.ScrollY != 0 && e.Frame.Contains(m.X, m.Y) {
		maxScroll := f32max(0, float32(len(tf.lines()))*tf.lineHeight()-tf.viewH())
		if maxScroll > 0 {
			tf.scrollY = clampf(tf.scrollY-m.ScrollY*WheelPixelsPerUnit, 0, maxScroll)
			m.ScrollY = 0
			m.Consumed = true
		}
	}
	if m.RightPressed && e.Frame.Contains(m.X, m.Y) {
		// Keep a selection the click lands in, so the menu can act on it;
		// anywhere else moves the caret there first.
		off := tf.offsetAt(m.X, m.Y)
		if lo, hi := tf.selRange(); !tf.hasSelection() || off < lo || off > hi {
			tf.moveTo(off, false)
		}
		if tf.openContextMenu(m.X, m.Y) {
			m.Consumed = true
		}
		return
	}
	if m.Pressed && e.Frame.Contains(m.X, m.Y) {
		off := tf.offsetAt(m.X, m.Y)
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
		tf.lastMerge = mergeNone
		tf.goalX = -1

		switch tf.clickCount {
		case 2: // word selection; a password is one word, so its spaces stay hidden
			lo, hi := wordRangeIn(tf.Value, off)
			if tf.cfg.Password {
				lo, hi = 0, len(tf.Value)
			}
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
		tf.caret = tf.offsetAt(m.X, m.Y)
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
	if !tf.focused || len(runes) == 0 || tf.disabled {
		return
	}
	s := string(runes)
	tf.edit(mergeType, s, func() { tf.insertAtCaret(s) })
	tf.refreshSuggest()
}

// ── Builder/modifier methods ─────────────────────────────────────────────────

// Changed sets the OnChange callback.
func (tf *TextInput) Changed(fn func(string)) *TextInput { tf.OnChange = fn; return tf }

// WithIconStart sets the leading icon.
func (tf *TextInput) WithIconStart(icon icons.Icon) *TextInput { tf.cfg.IconStart = icon; return tf }

// WithIconEnd sets the trailing icon.
func (tf *TextInput) WithIconEnd(icon icons.Icon) *TextInput { tf.cfg.IconEnd = icon; return tf }

// noCopy reports whether the field keeps its value off the clipboard.
func (tf *TextInput) noCopy() bool { return tf.cfg.Password && !tf.cfg.CopyMasked }

// AsPassword enables password masking (displays bullets instead of characters).
func (tf *TextInput) AsPassword() *TextInput { tf.cfg.Password = true; return tf }

// SelectAll selects the whole value.
func (tf *TextInput) SelectAll() {
	if tf.disabled {
		return
	}
	tf.selAnchor = 0
	tf.caret = len(tf.Value)
}

// SetSelection selects the byte range from anchor to caret, clamped to the
// value; the caret ends up at caret. Equal offsets just place the caret.
func (tf *TextInput) SetSelection(anchor, caret int) {
	if tf.disabled {
		return
	}
	clamp := func(i int) int { return max(0, min(i, len(tf.Value))) }
	tf.selAnchor = clamp(anchor)
	tf.caret = clamp(caret)
}

// HasSelection reports whether a non-empty range is selected.
func (tf *TextInput) HasSelection() bool { return tf.hasSelection() }

// Copy puts the selection on the clipboard, or the whole value when nothing is
// selected, and reports whether it copied anything. A password field never
// copies.
func (tf *TextInput) Copy() bool {
	clip := frameClipboard()
	if tf.disabled || tf.noCopy() || clip == nil {
		return false
	}
	if lo, hi := tf.selRange(); lo != hi {
		clip.Set(tf.Value[lo:hi])
		return true
	}
	if tf.Value != "" {
		clip.Set(tf.Value)
		return true
	}
	return false
}

// Cut moves the selection to the clipboard, or the whole value when nothing is
// selected. A password field ignores it, as it never copies.
func (tf *TextInput) Cut() {
	clip := frameClipboard()
	if tf.disabled || tf.noCopy() || clip == nil {
		return
	}
	if tf.hasSelection() {
		lo, hi := tf.selRange()
		clip.Set(tf.Value[lo:hi])
		tf.edit(mergeNone, "", func() { tf.deleteSelection() })
	} else if tf.Value != "" {
		clip.Set(tf.Value)
		tf.edit(mergeNone, "", func() {
			tf.selAnchor = -1
			tf.caret = 0
			tf.setValue("")
		})
	}
}

// Paste replaces the selection with the clipboard text: its first line, unless
// the field is multi-line.
func (tf *TextInput) Paste() {
	clip := frameClipboard()
	if tf.disabled || clip == nil {
		return
	}
	s := clip.Get()
	tf.edit(mergeNone, "", func() { tf.insertAtCaret(s) })
}

// openContextMenu shows the right-click menu at (x, y) and reports whether
// there was one to show.
func (tf *TextInput) openContextMenu(x, y float32) bool {
	if tf.disabled {
		return false
	}
	items := editMenuItems(tf, editMenuFlags{editable: true, copyable: !tf.noCopy()})
	if tf.ContextMenu != nil {
		items = tf.ContextMenu(items)
	}
	return tf.menu.open(items, x, y)
}

// HandleKeys processes navigation and editing keys for this frame.
func (tf *TextInput) HandleKeys(keys []input.KeyEvent) {
	if !tf.focused || tf.disabled {
		return
	}
	for _, ev := range keys {
		shift := ev.Mods.Has(input.ModShift)
		// An open completion popup takes navigation and acceptance keys first,
		// so Enter picks a suggestion rather than submitting the field.
		if tf.handleSuggestKey(ev) {
			continue
		}
		if ev.Key == input.KeyEnter && !ev.Mods.Primary() {
			if tf.cfg.Multiline && (shift || tf.OnSubmit == nil) {
				tf.edit(mergeNone, "", func() { tf.insertAtCaret("\n") })
				tf.refreshSuggest()
				continue
			}
			if tf.OnSubmit != nil {
				tf.OnSubmit(tf.Value)
			}
			continue
		}
		if ev.Mods.Primary() {
			switch ev.Key {
			case input.KeyUp, input.KeyHome:
				tf.moveTo(0, shift)
			case input.KeyDown, input.KeyEnd:
				tf.moveTo(len(tf.Value), shift)
			case input.KeyA:
				tf.SelectAll()
			case input.KeyC:
				tf.Copy()
			case input.KeyX:
				tf.Cut()
			case input.KeyV:
				tf.Paste()
			case input.KeyZ:
				if shift {
					tf.Redo()
				} else {
					tf.Undo()
				}
			case input.KeyY:
				tf.Redo()
			}
			continue
		}
		switch ev.Key {
		case input.KeyBackspace:
			if tf.hasSelection() {
				tf.edit(mergeNone, "", func() { tf.deleteSelection() })
			} else if tf.caret > 0 {
				tf.edit(mergeBackspace, "", func() {
					prev := prevRuneOff(tf.Value, tf.caret)
					tf.setValue(tf.Value[:prev] + tf.Value[tf.caret:])
					tf.caret = prev
				})
			}
			tf.refreshSuggest()
		case input.KeyDelete:
			if tf.hasSelection() {
				tf.edit(mergeNone, "", func() { tf.deleteSelection() })
			} else if tf.caret < len(tf.Value) {
				tf.edit(mergeDelete, "", func() {
					next := nextRuneOff(tf.Value, tf.caret)
					tf.setValue(tf.Value[:tf.caret] + tf.Value[next:])
				})
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
		case input.KeyUp, input.KeyDown:
			if tf.cfg.Multiline {
				delta := 1
				if ev.Key == input.KeyUp {
					delta = -1
				}
				tf.moveLine(delta, shift)
			}
		case input.KeyHome:
			if tf.cfg.Multiline {
				lo, _ := tf.lineBounds()
				tf.moveTo(lo, shift)
			} else {
				tf.moveTo(0, shift)
			}
		case input.KeyEnd:
			if tf.cfg.Multiline {
				_, hi := tf.lineBounds()
				tf.moveTo(hi, shift)
			} else {
				tf.moveTo(len(tf.Value), shift)
			}
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
