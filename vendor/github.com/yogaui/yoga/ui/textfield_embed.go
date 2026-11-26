package ui

import (
	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/textedit"
)

// A TextInput can live inside another widget's painting instead of the
// layout tree, as the fields of a code editor's find bar do. The host lays
// the field out itself, forwards the pointer with HandleMouseIn, paints it
// with PaintIn and runs LayoutHosted once a frame; Focus, Blur, HandleText
// and HandleKeys work as usual.

// FieldEmbed styles a hosted TextInput. The host draws the box around the
// field; PadRight keeps room on the right for the host's own buttons.
type FieldEmbed struct {
	PadRight   float32
	Background render.Color
	Border     render.Color
	Foreground render.Color // zero keeps the theme's text color
}

// Embed marks the field as hosted and sets how it is drawn.
func (tf *TextInput) Embed(e FieldEmbed) {
	tf.embed = fieldEmbed{
		on: true, padR: e.PadRight, bg: e.Background, line: e.Border,
		fg: e.Foreground, hasFg: e.Foreground.A > 0,
	}
}

// PaintIn paints the field into frame.
func (tf *TextInput) PaintIn(dl *render.DrawList, frame render.Rect) {
	tf.host.Frame = frame
	tf.paint(dl, nil)
}

// HandleMouseIn gives the field the pointer as if it sat at frame.
func (tf *TextInput) HandleMouseIn(frame render.Rect, m *input.Mouse) {
	tf.host.Frame = frame
	tf.onMouse(tf.host, m)
}

// Dragging reports whether a selection drag that started in the field is
// still going; the host keeps sending it the pointer until it ends.
func (tf *TextInput) Dragging() bool { return tf.dragging }

// LayoutHosted runs the field's per-frame upkeep, including its right-click
// menu. Call it from the host's Layout.
func (tf *TextInput) LayoutHosted(c *Ctx) {
	tf.Update(nil)
	tf.menu.Layout(c)
}

// ReplaceValue sets the value as one undoable edit, unlike assigning Value,
// which the field treats as the app resetting it.
func (tf *TextInput) ReplaceValue(s string) {
	tf.edit(textedit.MergeNone, "", func() { tf.setValue(s) })
}

// Selection returns the selected byte range, lo <= hi; lo == hi is the caret.
func (tf *TextInput) Selection() (lo, hi int) { return tf.selRange() }

// TextX is where the field's text starts, after its padding and start icon.
func (tf *TextInput) TextX() float32 { return tf.textLeft() }
