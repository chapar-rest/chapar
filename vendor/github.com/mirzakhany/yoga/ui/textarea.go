package ui

import (
	"math"
	"strings"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// textAreaData holds TextArea-only options.
type textAreaData struct {
	minRows, maxRows int
}

// Default row range of a TextArea without Rows.
const (
	textAreaMinRows = 3
	textAreaMaxRows = 10
)

// TextArea is a multi-line text input. Long lines wrap at the field's width
// and the height follows the line count within Rows, scrolling past the
// maximum. Enter inserts a newline, unless the field has OnSubmit: then Enter
// submits and Shift+Enter inserts the newline, as in a chat composer.
//
// It takes the TextField modifiers (Placeholder, OnChange, OnSubmit, Suggest,
// Highlight, DefaultFocus, Disabled, Style). The placeholder stays visible
// while the empty field is focused. value is controlled like a TextField's.
func TextArea(id, value string) *Node {
	return &Node{kind: kindTextArea, id: id, text: value,
		extra: &textAreaData{minRows: textAreaMinRows, maxRows: textAreaMaxRows}}
}

// Rows sets how many lines a TextArea shows: it starts at minRows and grows
// with its text up to maxRows, then scrolls.
func (n *Node) Rows(minRows, maxRows int) *Node {
	if d, ok := n.extra.(*textAreaData); ok {
		d.minRows = max(minRows, 1)
		d.maxRows = max(maxRows, d.minRows)
	}
	return n
}

func (n *Node) layoutTextArea(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "textarea")
	}
	d, _ := n.extra.(*textAreaData)
	if d == nil {
		d = &textAreaData{minRows: textAreaMinRows, maxRows: textAreaMaxRows}
	}
	placeholder := n.placeholder
	tf := c.Widget(id, func() any {
		return NewTextInput(TextFieldConfig{Placeholder: placeholder, Multiline: true})
	}).(*TextInput)
	tf.cfg.Placeholder = placeholder
	tf.cfg.MinRows, tf.cfg.MaxRows = d.minRows, d.maxRows
	n.bindTextInput(c, tf)

	if n.spec.hasH {
		tf.setHeight(n.spec.height)
	} else {
		tf.setHeight(tf.rowsHeight())
		tf.host.AfterLayout = func(e *layout.Element) bool {
			w, _ := e.LayoutSize()
			h := tf.rowsHeightAt(w)
			if h == e.Style.Height {
				return false
			}
			tf.setHeight(h)
			// One relayout per frame; ask for another in case this was it.
			c.Invalidate()
			return true
		}
	}
	return n.finishTextInput(c, tf)
}

// setHeight sizes the field's host.
func (tf *TextInput) setHeight(h float32) {
	tf.cfg.Height = h
	tf.host.Style.Height = h
	tf.host.Style.MinHeight = h
}

// lineHeight is the height of one line of body text.
func (tf *TextInput) lineHeight() float32 {
	th := theme.Current()
	if eng := frameText(); eng != nil {
		_, h := eng.MeasureAt("Ag", th.Typography.Body.Size)
		return h
	}
	return th.Typography.Body.Size
}

// padY centers one line in a control-height field, so a one-row area lines up
// with a TextField.
func (tf *TextInput) padY() float32 {
	return f32max(0, (theme.Current().Metrics.ControlHeight-tf.lineHeight())/2)
}

// wrapInset is the horizontal room the field takes from its text.
func (tf *TextInput) wrapInset() float32 {
	return tf.host.Frame.W - (tf.textRight() - tf.textLeft())
}

// rowsHeight is the height that shows the current lines within the row range.
func (tf *TextInput) rowsHeight() float32 {
	w := tf.host.Frame.W
	if w <= 0 {
		return tf.heightForRows(tf.cfg.MinRows)
	}
	return tf.rowsHeightAt(w)
}

func (tf *TextInput) rowsHeightAt(w float32) float32 {
	rows := len(tf.linesAt(w - tf.wrapInset()))
	rows = min(max(rows, tf.cfg.MinRows), tf.cfg.MaxRows)
	return tf.heightForRows(rows)
}

func (tf *TextInput) heightForRows(rows int) float32 {
	return float32(max(rows, 1))*tf.lineHeight() + 2*tf.padY()
}

// lines wraps the value at the text width. A width not known yet does not
// wrap.
func (tf *TextInput) lines() []textSpan {
	return tf.linesAt(tf.textRight() - tf.textLeft())
}

func (tf *TextInput) linesAt(w float32) []textSpan {
	if tf.spans != nil && tf.wrapW == w && tf.wrapText == tf.Value {
		return tf.spans
	}
	maxW := w
	if maxW <= 0 {
		maxW = float32(math.Inf(1))
	}
	tf.spans = wrapSpans(frameText(), tf.Value, theme.Current().Typography.Body.Size, shape.WeightRegular, maxW)
	tf.wrapW, tf.wrapText = w, tf.Value
	return tf.spans
}

// lineOf returns the index of the line holding off. At a soft wrap the
// offset belongs to the start of the next line.
func lineOf(spans []textSpan, off int) int {
	for i, sp := range spans {
		if off > sp.hi {
			continue
		}
		if off == sp.hi && i+1 < len(spans) && spans[i+1].lo == sp.hi {
			continue
		}
		return i
	}
	return len(spans) - 1
}

// textTop is the y of the first line, before scrolling.
func (tf *TextInput) textTop() float32 { return tf.host.Frame.Y + tf.padY() }

// viewH is the height of the visible text.
func (tf *TextInput) viewH() float32 { return f32max(0, tf.host.Frame.H-2*tf.padY()) }

// caretPos returns the top-left of the byte offset off, on screen.
func (tf *TextInput) caretPos(off int) (x, y float32) {
	spans := tf.lines()
	i := lineOf(spans, off)
	sp := spans[i]
	off = min(max(off, sp.lo), sp.hi)
	w, _ := frameText().MeasureAt(tf.Value[sp.lo:off], theme.Current().Typography.Body.Size)
	return tf.textLeft() + w, tf.textTop() - tf.scrollY + float32(i)*tf.lineHeight()
}

// offsetAt maps a point to the nearest byte offset.
func (tf *TextInput) offsetAt(px, py float32) int {
	if !tf.cfg.Multiline {
		return tf.offsetAtX(px)
	}
	spans := tf.lines()
	i := int(math.Floor(float64((py - tf.textTop() + tf.scrollY) / tf.lineHeight())))
	i = min(max(i, 0), len(spans)-1)
	return tf.offsetInLine(spans, i, px)
}

// offsetInLine maps x on line i to a byte offset. A soft-wrapped line ends
// before its trailing space, so the caret stays on the line clicked.
func (tf *TextInput) offsetInLine(spans []textSpan, i int, px float32) int {
	sp := spans[i]
	ln := frameText().LineAt(tf.Value[sp.lo:sp.hi], theme.Current().Typography.Body.Size)
	off := sp.lo + ln.ByteForX(px-tf.textLeft())
	if off == sp.hi && i+1 < len(spans) && spans[i+1].lo == sp.hi && off > sp.lo {
		off = prevRuneOff(tf.Value, off)
	}
	return off
}

// moveLine moves the caret delta lines up or down, keeping its column.
func (tf *TextInput) moveLine(delta int, extend bool) {
	spans := tf.lines()
	i := lineOf(spans, tf.caret)
	if tf.goalX < 0 {
		tf.goalX, _ = tf.caretPos(tf.caret)
	}
	goal := tf.goalX
	j := i + delta
	switch {
	case j < 0:
		tf.moveTo(0, extend)
	case j >= len(spans):
		tf.moveTo(len(tf.Value), extend)
	default:
		tf.moveTo(tf.offsetInLine(spans, j, goal), extend)
	}
	tf.goalX = goal
}

// lineBounds returns the start and end of the visual line holding the caret.
func (tf *TextInput) lineBounds() (lo, hi int) {
	spans := tf.lines()
	sp := spans[lineOf(spans, tf.caret)]
	return sp.lo, sp.hi
}

// followCaret scrolls so the caret's line is visible, after the caret or the
// value changed; wheel scrolling is left alone otherwise.
func (tf *TextInput) followCaret() {
	lh := tf.lineHeight()
	total := float32(len(tf.lines())) * lh
	maxScroll := f32max(0, total-tf.viewH())
	if tf.caret != tf.seenCaret || tf.Value != tf.seenValue {
		tf.seenCaret, tf.seenValue = tf.caret, tf.Value
		cy := float32(lineOf(tf.lines(), tf.caret)) * lh
		if cy < tf.scrollY {
			tf.scrollY = cy
		} else if cy+lh > tf.scrollY+tf.viewH() {
			tf.scrollY = cy + lh - tf.viewH()
		}
	}
	tf.scrollY = clampf(tf.scrollY, 0, maxScroll)
}

// paintLines draws a multi-line value, or the placeholder, with the selection
// and caret.
func (tf *TextInput) paintLines(dl *render.DrawList, col render.Color) {
	th := theme.Current()
	text := frameText()
	size := th.Typography.Body.Size
	f := tf.host.Frame
	tx, tr := tf.textLeft(), tf.textRight()
	lh := tf.lineHeight()
	clip := render.Rect{X: f.X, Y: f.Y + tf.padY(), W: f.W, H: tf.viewH()}
	tf.followCaret()

	dl.PushClip(clip)
	defer dl.PopClip()
	if tf.Value == "" {
		if tf.cfg.Placeholder != "" {
			pc := th.ForegroundMuted
			if tf.disabled {
				pc = th.ForegroundDisabled
			}
			y := tf.textTop()
			for _, line := range wrapText(text, tf.cfg.Placeholder, size, tr-tx) {
				text.DrawStringTopAt(dl, line, tx, y, pc, size)
				y += lh
			}
		}
	} else {
		spans := tf.lines()
		var hl []TextSpan
		if tf.Highlight != nil && !tf.cfg.Password {
			hl = clampSpans(tf.Highlight(tf.Value), len(tf.Value))
		}
		selLo, selHi := tf.selRange()
		y0 := tf.textTop() - tf.scrollY
		for i, sp := range spans {
			y := y0 + float32(i)*lh
			if y+lh < clip.Y || y > clip.Y+clip.H {
				continue
			}
			if tf.focused && selLo != selHi && selLo <= sp.hi && selHi >= sp.lo {
				a, b := max(selLo, sp.lo), min(selHi, sp.hi)
				xa, _ := text.MeasureAt(tf.Value[sp.lo:a], size)
				xb, _ := text.MeasureAt(tf.Value[sp.lo:b], size)
				if selHi > sp.hi && i+1 < len(spans) {
					xb += th.Stroke.Thick * 2 // the line break is selected too
				}
				if xb > xa {
					dl.AddRect(render.Rect{X: tx + xa, Y: y, W: xb - xa, H: lh}, th.Selection)
				}
			}
			line := tf.Value[sp.lo:sp.hi]
			if spans := lineSpans(hl, sp); len(spans) > 0 {
				drawSpannedText(dl, text, line, spans, tx, y, col, size)
			} else {
				text.DrawStringTopAt(dl, line, tx, y, col, size)
			}
		}
	}
	if tf.focused && tf.caretShown && !tf.disabled {
		cx, cy := tf.caretPos(tf.caret)
		dl.AddRect(render.Rect{X: clampf(cx, tx, tr), Y: cy, W: th.Stroke.Thick, H: lh}, th.Accent)
	}
}

// lineSpans shifts the highlight spans that fall on line sp to line offsets.
func lineSpans(spans []TextSpan, sp textSpan) []TextSpan {
	var out []TextSpan
	for _, s := range spans {
		lo, hi := max(s.Start, sp.lo), min(s.End, sp.hi)
		if lo < hi {
			out = append(out, TextSpan{Start: lo - sp.lo, End: hi - sp.lo, Color: s.Color})
		}
	}
	return out
}

// normalizeNewlines turns CRLF and lone CR into LF.
func normalizeNewlines(s string) string {
	if !strings.ContainsRune(s, '\r') {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
