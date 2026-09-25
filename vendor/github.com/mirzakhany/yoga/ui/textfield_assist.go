package ui

import (
	"sort"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TextSpan colors the byte range [Start,End) of a text field's value, so a
// field can mark up its own text — a template placeholder, a match, a token.
type TextSpan struct {
	Start, End int
	Color      render.Color
}

// Suggestion is one entry in a text field's completion popup.
type Suggestion struct {
	// Label is the row's text and, unless Insert is set, the text inserted.
	Label string
	// Detail is a dim hint painted right-aligned, such as a variable's value.
	Detail string
	// Insert replaces the suggested range when it differs from Label.
	Insert string
}

// insertText is what accepting the suggestion writes.
func (s Suggestion) insertText() string {
	if s.Insert != "" {
		return s.Insert
	}
	return s.Label
}

// SuggestFunc offers completions for a field. It receives the current value and
// the caret's byte offset, and returns the suggestions plus the byte range they
// replace. Returning no suggestions closes the popup.
type SuggestFunc func(value string, caret int) (items []Suggestion, start, end int)

// suggestState is a field's completion popup: the menu that paints it and the
// range the next accepted suggestion replaces.
type suggestState struct {
	menu       *Menu
	start, end int
}

// suggestPopupMinWidth keeps a narrow field's popup readable.
const suggestPopupMinWidth = 240

// Highlighted sets the function that colors ranges of the value. It is asked
// once per paint, so it should be cheap.
func (tf *TextInput) Highlighted(fn func(value string) []TextSpan) *TextInput {
	tf.Highlight = fn
	return tf
}

// Suggests sets the completion source. The popup opens after edits that leave
// the caret somewhere the source offers suggestions for.
func (tf *TextInput) Suggests(fn SuggestFunc) *TextInput {
	tf.Suggest = fn
	return tf
}

// SuggestOpen reports whether the completion popup is showing.
func (tf *TextInput) SuggestOpen() bool {
	return tf.sugg.menu != nil && tf.sugg.menu.Open
}

// RegisterSuggest registers the completion popup as a frame overlay when it is
// open. Widgets that own a TextInput call it after laying the field out.
func (tf *TextInput) RegisterSuggest(c *Ctx) {
	if !tf.SuggestOpen() {
		return
	}
	tf.sugg.menu.BindPaint(c)
	c.Overlay(tf.sugg.menu.overlay())
}

// closeSuggest hides the popup.
func (tf *TextInput) closeSuggest() {
	if tf.sugg.menu != nil {
		tf.sugg.menu.Close()
	}
}

// refreshSuggest asks the source for completions at the caret and opens, moves,
// or closes the popup to match.
func (tf *TextInput) refreshSuggest() {
	if tf.Suggest == nil || !tf.focused || tf.disabled || tf.cfg.Password {
		tf.closeSuggest()
		return
	}
	items, start, end := tf.Suggest(tf.Value, tf.caret)
	if len(items) == 0 || start < 0 || end > len(tf.Value) || start > end {
		tf.closeSuggest()
		return
	}
	tf.sugg.start, tf.sugg.end = start, end
	entries := make([]MenuItem, len(items))
	for i, it := range items {
		text := it.insertText()
		entries[i] = MenuItem{Label: it.Label, Shortcut: it.Detail, OnSelect: func() {
			tf.applySuggestion(text)
		}}
	}
	w := tf.host.Frame.W
	if w < suggestPopupMinWidth {
		w = suggestPopupMinWidth
	}
	if tf.sugg.menu == nil {
		tf.sugg.menu = NewMenu(w, entries)
	} else {
		tf.sugg.menu.width = w
		tf.sugg.menu.SetItems(entries)
	}
	f := tf.host.Frame
	x := tf.textLeft() - tf.scrollX + tf.offsetXForValue(start)
	tf.sugg.menu.OpenAt(clampf(x, f.X, f.X+f.W), tf.suggestY())
	// OpenAt leaves the hover on a checked item; a fresh list starts at the top.
	tf.sugg.menu.hover = 0
	tf.sugg.menu.scrollY = 0
}

// suggestY is where the popup opens: under the field, or above it when the room
// below is too tight and there is more of it above. Either way the popup must
// not cover the text being completed.
func (tf *TextInput) suggestY() float32 {
	f := tf.host.Frame
	gap := theme.Current().Spacing.XS
	below := f.Y + f.H + gap
	if viewportH <= 0 {
		return below
	}
	h := tf.sugg.menu.height()
	roomBelow := viewportH - below
	roomAbove := f.Y - gap
	if h <= roomBelow || roomAbove <= roomBelow {
		return below
	}
	return f32max(0, f.Y-gap-f32min(h, roomAbove))
}

// applySuggestion replaces the suggested range with text and puts the caret
// after it.
func (tf *TextInput) applySuggestion(text string) {
	start, end := tf.sugg.start, tf.sugg.end
	if start < 0 || end > len(tf.Value) || start > end {
		tf.closeSuggest()
		return
	}
	tf.closeSuggest()
	tf.edit(mergeNone, "", func() {
		tf.selAnchor = -1
		tf.setValue(tf.Value[:start] + text + tf.Value[end:])
		tf.caret = start + len(text)
		tf.clampCaret()
	})
}

// handleSuggestKey lets the open popup take the navigation keys. It reports
// whether the key was consumed.
func (tf *TextInput) handleSuggestKey(ev input.KeyEvent) bool {
	if !tf.SuggestOpen() || ev.Mods.Primary() {
		return false
	}
	switch ev.Key {
	case input.KeyDown:
		tf.sugg.menu.MoveHover(1)
		return true
	case input.KeyUp:
		tf.sugg.menu.MoveHover(-1)
		return true
	case input.KeyEnter, input.KeyTab:
		return tf.sugg.menu.ActivateHover()
	case input.KeyEscape:
		tf.closeSuggest()
		return true
	case input.KeyLeft, input.KeyRight, input.KeyHome, input.KeyEnd:
		tf.closeSuggest()
		return false
	}
	return false
}

// highlightSpans returns the spans to color for the text about to be painted.
// Masked values and the placeholder are painted plain: their offsets are not
// the value's.
func (tf *TextInput) highlightSpans(show string) []TextSpan {
	if tf.Highlight == nil || tf.cfg.Password || show != tf.Value {
		return nil
	}
	return clampSpans(tf.Highlight(tf.Value), len(tf.Value))
}

// clampSpans drops spans outside [0,n) or inverted, sorts what is left, and
// keeps the first of any two that overlap, so drawing them walks the text once.
func clampSpans(spans []TextSpan, n int) []TextSpan {
	if len(spans) == 0 {
		return nil
	}
	out := make([]TextSpan, 0, len(spans))
	for _, sp := range spans {
		if sp.Start < 0 || sp.End > n || sp.Start >= sp.End {
			continue
		}
		out = append(out, sp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	kept := out[:0]
	end := 0
	for _, sp := range out {
		if sp.Start < end {
			continue
		}
		kept = append(kept, sp)
		end = sp.End
	}
	return kept
}

// drawSpannedText paints show with the given spans in their own colors and the
// rest in base. x is the left edge of the whole string, before any span.
func drawSpannedText(dl *render.DrawList, text *shape.Engine, show string, spans []TextSpan, x, y float32, base render.Color, size float32) {
	at := 0
	for _, sp := range spans {
		if sp.Start > at {
			seg := show[at:sp.Start]
			text.DrawStringTopAt(dl, seg, x, y, base, size)
			w, _ := text.MeasureAt(seg, size)
			x += w
		}
		seg := show[sp.Start:sp.End]
		text.DrawStringTopAt(dl, seg, x, y, sp.Color, size)
		w, _ := text.MeasureAt(seg, size)
		x += w
		at = sp.End
	}
	if at < len(show) {
		text.DrawStringTopAt(dl, show[at:], x, y, base, size)
	}
}

// HoverCard is what a field shows when the pointer rests over part of its
// value: a title, a body, and the byte range they describe, which the field
// uses to place the card under the words it explains.
type HoverCard struct {
	Title, Body string
	Start, End  int
}

// HoverInfoFunc is asked what to show for the byte offset under the pointer.
// Returning false shows nothing.
type HoverInfoFunc func(value string, off int) (HoverCard, bool)

// hoverState tracks the pointer resting over part of a field's value. The zero
// value means the pointer has not been seen yet, which is why off is set to -1
// when a field is built: offset 0 is a real place in the text.
type hoverState struct {
	off   int // byte offset under the pointer, or -1 for none
	since time.Time
	shown bool
}

// Explained sets the hover source: what to show when the pointer rests over
// part of the value.
func (tf *TextInput) Explained(fn HoverInfoFunc) *TextInput {
	tf.HoverInfo = fn
	return tf
}

// trackHover notes where the pointer is, so the card can appear once it has
// rested over the same part of the value.
func (tf *TextInput) trackHover(e *layout.Element, m *input.Mouse) {
	if tf.HoverInfo == nil || tf.cfg.Password || m == nil {
		return
	}
	off := -1
	if !m.Down && e.Frame.Contains(m.X, m.Y) && m.X >= tf.textLeft() && m.X <= tf.textRight() {
		off = tf.offsetAtX(m.X)
	}
	if off != tf.hover.off || tf.hover.since.IsZero() {
		tf.hover = hoverState{off: off, since: time.Now()}
	}
}

// layoutHoverCard registers the hover card overlay once the pointer has rested
// long enough over a part of the value the source explains.
func (tf *TextInput) layoutHoverCard(c *Ctx) {
	if tf.HoverInfo == nil || tf.hover.off < 0 || tf.cfg.Password {
		tf.hover.shown = false
		return
	}
	card, ok := tf.HoverInfo(tf.Value, tf.hover.off)
	if !ok || (card.Title == "" && card.Body == "") {
		tf.hover.shown = false
		return
	}
	if !tf.hover.shown {
		if rest := c.Now().Sub(tf.hover.since); rest < tooltipDelay {
			c.Animate(tooltipDelay - rest)
			return
		}
		tf.hover.shown = true
	}
	c.Overlay(buildHoverCardOverlay(c, tf.hoverAnchor(card), card))
}

// hoverAnchor is the rectangle the card points at: the hovered range, clamped
// to the visible part of the field.
func (tf *TextInput) hoverAnchor(card HoverCard) render.Rect {
	f := tf.host.Frame
	lo, hi := tf.textLeft(), tf.textRight()
	x0, x1 := lo, hi
	if card.End > card.Start && card.End <= len(tf.Value) {
		base := tf.textLeft() - tf.scrollX
		x0 = clampf(base+tf.offsetXForValue(card.Start), lo, hi)
		x1 = clampf(base+tf.offsetXForValue(card.End), lo, hi)
	}
	if x1 <= x0 {
		x1 = x0 + 1
	}
	return render.Rect{X: x0, Y: f.Y, W: x1 - x0, H: f.H}
}
