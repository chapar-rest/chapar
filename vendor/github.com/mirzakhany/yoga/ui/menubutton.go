package ui

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// MenuButton is a button-styled trigger that opens a menu of items. Without
// OnClick the whole control toggles the menu; with OnClick the label runs the
// action and the chevron toggles the menu (split button).
func MenuButton(id, label string, items []MenuItem) *Node {
	return &Node{kind: kindMenuButton, id: id, text: label, extra: &dropdownData{items: items}}
}

func (n *Node) layoutMenuButton(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "menubutton")
	}
	d, _ := n.extra.(*dropdownData)
	items := []MenuItem{}
	if d != nil {
		items = d.items
	}
	mst := c.Widget(id+"-menu", func() any { return &dropdownState{} }).(*dropdownState)
	st := c.Widget(id, func() any { return &buttonState{} }).(*buttonState)
	if n.disabled {
		st.hovered, st.pressed = false, false
	}
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	spec := n.buttonSpec(c)
	label := n.text
	icon := n.iconStart
	onClick := n.onClick
	disabled := n.disabled
	split := onClick != nil

	padX, _, iconGap, h := buttonMetrics(n, c)
	iconSz := th.Metrics.IconSizeSM
	chevronSlot := iconSz + padX
	padLeft, padRight := padX, padX+chevronSlot
	minW := padLeft + padRight
	if !icon.Empty() {
		iconSlot := iconSz + iconGap
		minW += iconSlot
		padLeft += iconSlot
	}
	if c.Text() != nil {
		lw, _ := c.Text().MeasureAt(label, th.Typography.Body.Size)
		minW += lw
	}
	if split {
		minW += th.Stroke.Thin
	}

	style := layout.Box().
		Direction(layout.Row).
		AlignItems(layout.AlignCenter).
		H(h).Min(minW, h).FlexShrink(0)
	style.Padding = layout.Edges{Left: padLeft, Right: padRight}
	box := applyLayoutSpec(style, spec)
	el := layout.New(box)
	st.el = el

	triggerW := minW
	if n.spec.hasW {
		triggerW = n.spec.width
	}
	menuW := float32(160)
	if triggerW > menuW {
		menuW = triggerW
	}
	if mst.menu == nil {
		mst.menu = NewMenu(menuW, items)
	} else {
		mst.menu.SetItems(items)
		mst.menu.width = menuW
	}

	toggleMenu := func() {
		if mst.menu.Open {
			mst.menu.Close()
			return
		}
		f := el.Frame
		mst.menu.width = triggerMenuWidth(menuW, f.W)
		mst.menu.OpenAt(f.X, f.Y+f.H)
	}

	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		r := spec.resolve(th, interactState{
			hovered: st.hovered, pressed: st.pressed, focused: st.focused, disabled: disabled,
		})
		frame := scaledFrame(el.Frame, r.scaleX, r.scaleY)
		paintResolvedBox(dl, frame, r, th.Radius.Medium)
		if st.focused {
			fill := r.bg
			if fill.A == 0 {
				fill = th.Chrome
			}
			paintFocusRing(dl, el.Frame, fill, th)
		}
		fg := r.fg
		if !r.hasFg {
			fg = th.Foreground
		}
		cy := frame.Y + frame.H/2
		x := frame.X + padX
		if !icon.Empty() {
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, icon, render.Rect{X: x, Y: cy - iconSz/2, W: iconSz, H: iconSz}, fg)
			}
			x += iconSz + iconGap
		}
		style := th.Typography.Body
		_, lh := text.MeasureAt(label, style.Size)
		text.DrawStringTopAt(dl, label, x, cy-lh/2, fg, style.Size)

		chevronLeft := frame.X + frame.W - chevronSlot
		if split {
			divX := chevronLeft - th.Stroke.Thin
			divH := frame.H * 0.55
			divY := frame.Y + (frame.H-divH)/2
			dl.AddRect(render.Rect{X: divX, Y: divY, W: th.Stroke.Thin, H: divH}, th.Border)
		}
		chevCol := th.ForegroundMuted
		if r.hasFg {
			chevCol = render.Color{R: fg.R, G: fg.G, B: fg.B, A: fg.A * 0.72}
		}
		ix := chevronLeft + (chevronSlot-iconSz-padX/2)/2
		if ix < chevronLeft {
			ix = chevronLeft + padX/4
		}
		iy := cy - iconSz/2
		if sheet := frameIcons(); sheet != nil {
			sheet.Draw(dl, icons.ChevronDown, render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, chevCol)
		}
	}

	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if disabled {
			return
		}
		inside := e.Frame.Contains(m.X, m.Y)
		trackHover(c, &st.hovered, inside)
		if inside {
			m.SetCursor(CursorPointer)
		}
		chevronLeft := e.Frame.X + e.Frame.W - chevronSlot
		inChevron := inside && m.X >= chevronLeft
		if inside && m.Pressed {
			trackBool(c, &st.pressed, true)
			m.Consumed = true
		}
		if m.Released {
			if st.pressed && inside {
				if split {
					if inChevron {
						toggleMenu()
					} else if onClick != nil {
						onClick()
					}
				} else {
					toggleMenu()
				}
				c.MarkNeedsPaint()
			}
			trackBool(c, &st.pressed, false)
		}
	}

	if mst.menu.Open {
		mst.menu.BindPaint(c)
		c.Overlay(mst.menu.overlay())
	}
	return el
}
