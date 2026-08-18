package ui

import (
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
	h := th.Metrics.ControlHeight
	w := n.spec.width
	if !n.spec.hasW {
		w = 160
	}
	el := layout.New(applyLayoutSpec(layout.Box().W(w).H(h).FlexShrink(0), n.spec))
	st.el = el

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
		bg := th.ChromeMuted
		if st.hovered || st.pressed {
			bg = th.ListHover
		}
		border := th.Border
		if st.focused {
			border = th.FocusRing
		}
		dl.AddRoundedRectBorder(f, th.Radius.Medium, th.Stroke.Thin, bg, border)
		labelCol := th.Foreground
		if hasTint {
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
			sheet.Draw(dl, "expand_more", render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, th.ForegroundMuted)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		st.hovered = e.Frame.Contains(m.X, m.Y)
		if st.hovered {
			m.SetCursor(CursorPointer)
		}
		if st.hovered && m.Pressed {
			st.pressed = true
			m.Consumed = true
		}
		if m.Released && st.pressed && st.hovered {
			if st.menu.Open {
				st.menu.Close()
			} else {
				fr := e.Frame
				st.menu.OpenAt(fr.X, fr.Y+fr.H)
			}
		}
		if !m.Down {
			st.pressed = false
		}
	}
	if st.menu.Open {
		c.Overlay(st.menu.overlay())
	}
	return el
}
