package ui

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// SelectOption is one entry in a Select control.
type SelectOption struct {
	Label string
	Value string
}

type selectData struct {
	options []SelectOption
	colors  map[string]render.Color
}

type selectState struct {
	hovered, pressed, focused bool
	menu                      *Menu
	el                        *layout.Element
}

func (s *selectState) Focus()                        { s.focused = true }
func (s *selectState) Blur()                         { s.focused = false }
func (s *selectState) Focused() bool                 { return s.focused }
func (s *selectState) FocusEl() *layout.Element      { return s.el }
func (s *selectState) FocusOnClick() bool            { return true }
func (s *selectState) CapturesTab() bool             { return false }
func (s *selectState) HandleText(_ []rune)           {}
func (s *selectState) HandleKeys(_ []input.KeyEvent) {}

// Select is a form dropdown. Selected index is controlled via .Selected(i).
func Select(id string, options []SelectOption) *Node {
	return &Node{kind: kindSelect, id: id, extra: &selectData{options: options}}
}

// OptionColor tints the trigger when the option with the given value is selected.
func (n *Node) OptionColor(value string, col render.Color) *Node {
	d, ok := n.extra.(*selectData)
	if !ok {
		return n
	}
	if d.colors == nil {
		d.colors = make(map[string]render.Color)
	}
	d.colors[value] = col
	return n
}

func (n *Node) layoutSelect(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "select")
	}
	d, _ := n.extra.(*selectData)
	if d == nil {
		d = &selectData{}
	}
	st := c.Widget(id, func() any { return &selectState{} }).(*selectState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	h := c.controlHeight()
	w := n.spec.width
	if !n.spec.hasW {
		w = 160
	}
	el := layout.New(applyLayoutSpec(layout.Box().W(w).H(h).FlexShrink(0), n.spec))
	st.el = el

	disabled := n.disabled
	suppressHoverPressIfDisabled(disabled, &st.hovered, &st.pressed)
	if disabled {
		if st.focused {
			st.Blur()
		}
		if st.menu != nil && st.menu.Open {
			st.menu.Close()
		}
	}
	spec := c.styles().Select.merge(n.spec)

	sel := n.selected
	if sel < 0 || sel >= len(d.options) {
		sel = 0
	}
	label := ""
	value := ""
	if sel < len(d.options) {
		label = d.options[sel].Label
		value = d.options[sel].Value
	}
	tint, hasTint := d.colors[value]
	onChange := n.onChange
	onSelectIdx := n.onSelectIdx
	options := d.options
	width := w

	if st.menu == nil {
		st.menu = NewMenu(width, nil)
	}
	items := make([]MenuItem, len(options))
	for i, opt := range options {
		i, opt := i, opt
		items[i] = MenuItem{Label: opt.Label, OnSelect: func() {
			if onSelectIdx != nil {
				onSelectIdx(i, opt.Value)
			}
			if onChange != nil {
				onChange(opt.Value)
			}
		}}
	}
	st.menu.SetItems(items)
	st.menu.width = width

	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		inter := interactStateFor(disabled, st.hovered, st.pressed, st.focused)
		r := spec.resolve(th, inter)
		bg := th.ChromeMuted
		if r.hasBg {
			bg = r.bg
		} else if st.hovered || st.pressed {
			bg = th.ListHover
		}
		border := th.Border
		if r.hasBorder {
			border = r.border
		} else if st.focused {
			border = th.FocusRing
		}
		radius := th.Radius.Medium
		if r.hasRadii {
			radius = uniformRadius(r.radii, th.Radius.Medium)
		}
		bw := uniformBorderWidth(r.borderW, th.Stroke.Thin)
		paintChromeBox(dl, f, r, bg, border, bw, radius)
		labelCol := th.Foreground
		if disabled {
			labelCol = th.ForegroundDisabled
		} else if r.hasFg {
			labelCol = r.fg
		}
		if hasTint && !disabled {
			dl.PushClip(f)
			dl.AddRect(render.Rect{X: f.X, Y: f.Y, W: 2, H: f.H}, tint)
			dl.PopClip()
			labelCol = tint
		}
		style := th.Typography.Body
		_, lh := text.MeasureAt(label, style.Size)
		pad := th.Spacing.MNudge
		text.DrawStringTopAt(dl, label, f.X+pad, f.Y+(f.H-lh)/2, labelCol, style.Size)
		iconSz := th.Metrics.IconSizeSM
		ix := f.X + f.W - pad - iconSz
		iy := f.Y + (f.H-iconSz)/2
		if sheet := frameIcons(); sheet != nil {
			chevCol := th.ForegroundMuted
			if disabled {
				chevCol = th.ForegroundDisabled
			}
			sheet.Draw(dl, icons.ChevronDown, render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, chevCol)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if disabled {
			return
		}
		trackHover(c, &st.hovered, e.Frame.Contains(m.X, m.Y))
		if st.hovered {
			m.SetCursor(CursorPointer)
		}
		if st.hovered && m.Pressed {
			trackBool(c, &st.pressed, true)
			m.Consumed = true
		}
		if m.Released && st.pressed && st.hovered {
			if st.menu.Open {
				st.menu.Close()
			} else {
				fr := e.Frame
				st.menu.width = triggerMenuWidth(width, fr.W)
				st.menu.OpenAt(fr.X, fr.Y+fr.H)
			}
			c.MarkNeedsPaint()
		}
		if !m.Down {
			trackBool(c, &st.pressed, false)
		}
	}
	if st.menu.Open {
		st.menu.BindPaint(c)
		c.Overlay(st.menu.overlay())
	}
	return el
}