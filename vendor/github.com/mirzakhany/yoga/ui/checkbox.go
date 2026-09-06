package ui

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type checkboxState struct {
	hovered, focused bool
	el               *layout.Element
	toggle           func()
}

func (b *checkboxState) Focus()                      { b.focused = true }
func (b *checkboxState) Blur()                       { b.focused = false }
func (b *checkboxState) Focused() bool               { return b.focused }
func (b *checkboxState) HandleKeys([]input.KeyEvent) {}
func (b *checkboxState) CapturesTab() bool           { return false }
func (b *checkboxState) FocusOnClick() bool          { return true }
func (b *checkboxState) FocusEl() *layout.Element    { return b.el }

func (b *checkboxState) HandleText(runes []rune) {
	if !b.focused {
		return
	}
	for _, r := range runes {
		if r == ' ' && b.toggle != nil {
			b.toggle()
		}
	}
}

var _ Focusable = (*checkboxState)(nil)

// Checkbox is a labeled toggle. id keys hover/focus; checked is controlled by the app.
func Checkbox(id, label string) *Node {
	return &Node{kind: kindCheckbox, id: id, text: label}
}

func (n *Node) layoutCheckbox(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "checkbox")
	}
	st := c.Widget(id, func() any { return &checkboxState{} }).(*checkboxState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	disabled := n.disabled
	if disabled {
		st.hovered = false
		if st.focused {
			st.Blur()
		}
	}

	th := c.Theme()
	box := th.Metrics.IconSizeSM
	style := th.Typography.Body
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	h := box
	if lh > h {
		h = lh
	}
	w := box + th.Spacing.S + tw
	el := layout.New(applyLayoutSpec(layout.Box().Size(w, h).FlexShrink(0), n.spec))
	st.el = el

	checked := n.checked
	onToggle := n.onToggle
	st.toggle = func() {
		if disabled {
			return
		}
		if onToggle != nil {
			onToggle(!checked)
		}
	}
	label := n.text
	muted := n.labelMuted
	strike := n.labelStrike
	spec := c.styles().Checkbox.merge(n.spec)

	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		bx := f.X
		by := f.Y + (f.H-box)/2
		br := render.Rect{X: bx, Y: by, W: box, H: box}
		inter := interactStateFor(disabled, st.hovered, false, st.focused)
		r := spec.resolve(th, inter)
		fill := th.Chrome
		if r.hasBg {
			fill = r.bg
		}
		border := th.Border
		if r.hasBorder {
			border = r.border
		}
		if checked && !disabled {
			fill = th.Accent
			border = th.Accent
			if st.hovered {
				fill = th.AccentHover
			}
		} else if st.hovered && !disabled {
			fill = th.ListHover
		}
		if st.focused && !disabled {
			border = th.FocusRing
		}
		radius := th.Radius.Small
		if r.hasRadii {
			radius = uniformRadius(r.radii, th.Radius.Small)
		}
		bw := uniformBorderWidth(r.borderW, th.Stroke.Thin)
		paintChromeBox(dl, br, r, fill, border, bw, radius)
		if checked {
			inner := render.Rect{X: bx + 2, Y: by + 2, W: box - 4, H: box - 4}
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, icons.Check, inner, th.AccentForeground)
			}
		}
		tx := bx + box + th.Spacing.S
		_, llh := text.MeasureAt(label, style.Size)
		ty := f.Y + (f.H-llh)/2
		col := th.Foreground
		if disabled {
			col = th.ForegroundDisabled
		} else if r.hasFg {
			col = r.fg
		}
		if muted {
			col = th.ForegroundMuted
		}
		text.DrawStringTopAt(dl, label, tx, ty, col, style.Size)
		if strike {
			ltw, _ := text.MeasureAt(label, style.Size)
			midY := ty + llh/2
			strikeCol := col
			if !muted {
				strikeCol = th.ForegroundMuted
			}
			dl.AddRect(render.Rect{X: tx, Y: midY - th.Stroke.Thin/2, W: ltw, H: th.Stroke.Thin}, strikeCol)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if disabled {
			return
		}
		st.hovered = e.Frame.Contains(m.X, m.Y)
		if st.hovered && m.Released {
			st.toggle()
			m.Consumed = true
		}
	}
	return el
}
