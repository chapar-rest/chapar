package ui

import (
	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/layout"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/shape"
)

type buttonState struct {
	hovered, pressed, focused bool
	el                        *layout.Element
}

func (b *buttonState) Focus()                      { b.focused = true }
func (b *buttonState) Blur()                       { b.focused = false }
func (b *buttonState) Focused() bool               { return b.focused }
func (b *buttonState) HandleText([]rune)           {}
func (b *buttonState) HandleKeys([]input.KeyEvent) {}
func (b *buttonState) CapturesTab() bool           { return false }
func (b *buttonState) FocusOnClick() bool          { return true }
func (b *buttonState) FocusEl() *layout.Element    { return b.el }

var _ Focusable = (*buttonState)(nil)

// Button is a clickable control. id keys hover/press/focus across frames.
// child is typically Text.
func Button(id string, child View) *Node {
	return &Node{kind: kindButton, id: id, child: child}
}

func (n *Node) buttonSpec(c *Ctx) Spec {
	st := c.styles()
	base := st.ButtonSecondary
	switch n.variant {
	case variantPrimary:
		base = st.ButtonPrimary
	case variantSubtle:
		base = st.ButtonSubtle
	case variantGhost:
		base = st.ButtonGhost
		if n.ghostHover {
			base = st.ButtonGhostHover
		}
	}
	return base.merge(n.spec)
}

// buttonMetrics returns padding, icon gap, and height for a button variant.
func buttonMetrics(n *Node, c *Ctx) (padX, padY, iconGap, h float32) {
	th := c.Theme()
	h = c.controlHeight()
	compact := c.env.hasControlHeight
	if n.variant == variantGhost {
		if n.ghostHover {
			padX = th.Spacing.XS
			padY = th.Spacing.XXS
			return padX, padY, th.Spacing.XS, th.Typography.Body.LineHeight + 2*padY
		}
		return 0, 0, th.Spacing.XS, th.Typography.Body.LineHeight
	}
	if compact {
		padX = th.Spacing.S
		padY = th.Spacing.XXS
	} else {
		padX = th.Spacing.M
		padY = th.Spacing.SNudge
	}
	return padX, padY, 8, h
}

func (n *Node) layoutButton(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "button")
	}
	st := c.Widget(id, func() any { return &buttonState{} }).(*buttonState)
	if n.disabled {
		st.hovered, st.pressed = false, false
	}
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	j := c.env.joined
	spec := n.buttonSpec(c)
	inter := interactState{hovered: st.hovered, pressed: st.pressed, focused: st.focused, focusVisible: focusVisible(st.focused), disabled: n.disabled}
	res := spec.resolve(th, inter)

	old := c.pushEnv(env{
		textColor: res.fg,
		hasColor:  res.hasFg,
		fontSize:  th.Typography.Body.Size,
		hasSize:   true,
	})
	var childEl *layout.Element
	if n.child != nil {
		childEl = n.child.Layout(c)
	}
	c.popEnv(old)

	padX, _, iconGap, h := buttonMetrics(n, c)
	minW := 2 * padX
	if childEl != nil && childEl.Style.Width == childEl.Style.Width {
		minW += float32(childEl.Style.Width)
	}
	padLeft, padRight := padX, padX
	if !n.iconStart.Empty() {
		iconSlot := th.Metrics.IconSizeSM + iconGap
		minW += iconSlot
		padLeft += iconSlot
	}
	if n.hint != "" && c.Text() != nil {
		hw, _ := c.Text().MeasureAt(n.hint, th.Typography.Caption.Size)
		hintSlot := th.Spacing.S + hw + th.Spacing.MNudge
		minW += hintSlot
		padRight += hintSlot
	}

	style := layout.Box().
		Direction(layout.Row).
		AlignItems(layout.AlignCenter).
		H(h).Min(minW, h).FlexShrink(0)
	style.Padding = layout.Edges{Left: padLeft, Right: padRight}
	box := applyLayoutSpec(style, spec)
	el := layout.New(box)
	if childEl != nil {
		el.Children = []*layout.Element{childEl}
	}
	st.el = el

	onClick := n.onClick
	disabled := n.disabled
	icon := n.iconStart
	hint := n.hint
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		r := spec.resolve(th, interactState{
			hovered: st.hovered, pressed: st.pressed, focused: st.focused, disabled: disabled,
			focusVisible: focusVisible(st.focused),
		})
		frame := scaledFrame(el.Frame, r.scaleX, r.scaleY)
		if j.in {
			fill := r.bg
			if !r.hasBg {
				fill = render.Color{}
			}
			paintJoinedFill(dl, el.Frame, j, fill, th.Radius.Medium)
		} else {
			paintResolvedBox(dl, frame, r, th.Radius.Medium)
		}
		if focusVisible(st.focused) && !j.in {
			paintFocusRing(dl, el.Frame, th.Radius.Medium, th)
		}
		fg := r.fg
		if !r.hasFg {
			fg = th.Foreground
		}
		x := frame.X + padX
		cy := frame.Y + frame.H/2
		if !icon.Empty() {
			isz := th.Metrics.IconSizeSM
			if sheet := FrameIcons(); sheet != nil {
				sheet.Draw(dl, icon, render.Rect{X: x, Y: cy - isz/2, W: isz, H: isz}, fg)
			}
		}
		if hint != "" {
			hw, hh := text.MeasureAt(hint, th.Typography.Caption.Size)
			chipW := hw + th.Spacing.MNudge
			chipH := hh + 2*th.Stroke.Thin
			chip := render.Rect{X: frame.X + frame.W - padX - chipW, Y: cy - chipH/2, W: chipW, H: chipH}
			// The chip is a light wash of the label color; the shortcut text
			// itself stays at full label color, which a 62% alpha did not
			// keep readable on lighter button fills.
			dl.AddRoundedRect(chip, th.Radius.Small, alpha(fg, 0.16))
			text.DrawStringTopAt(dl, hint, chip.X+(chipW-hw)/2, cy-hh/2, fg, th.Typography.Caption.Size)
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
		if inside && m.Pressed {
			trackBool(c, &st.pressed, true)
			m.Consumed = true
		}
		if m.Released {
			if st.pressed && inside && onClick != nil {
				onClick()
				c.MarkNeedsPaint()
			}
			trackBool(c, &st.pressed, false)
		}
	}
	return el
}
