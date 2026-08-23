package ui

import (
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

const tooltipDelay = 400 * time.Millisecond

type tooltipState struct {
	hovered bool
	hoverAt time.Time
	visible bool
	anchor  render.Rect
	text    string
}

// Tooltip wraps child and shows text after a hover delay.
func Tooltip(id string, child View, text string) *Node {
	n := ViewOf(child)
	if id != "" {
		n.id = id
	}
	n.tooltip = text
	return n
}

// Tooltip sets hover-hint text on any Node. Applied after the node's Layout.
func (n *Node) Tooltip(text string) *Node {
	n.tooltip = text
	return n
}

func attachNodeTooltip(c *Ctx, id, text string, el *layout.Element) {
	if el == nil || text == "" {
		return
	}
	if id == "" {
		id = autoID(c, "tooltip")
	}
	st := c.Widget(id+"#tip", func() any { return &tooltipState{} }).(*tooltipState)
	st.text = text

	prevMouse := el.OnMouse
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if prevMouse != nil {
			prevMouse(e, m)
		}
		inside := e.Frame.Contains(m.X, m.Y)
		now := c.Now()
		if inside {
			if !st.hovered {
				st.hovered = true
				st.hoverAt = now
				st.visible = false
			}
			st.anchor = e.Frame
			if !st.visible && now.Sub(st.hoverAt) >= tooltipDelay {
				st.visible = true
			}
		} else {
			st.hovered = false
			st.visible = false
		}
	}

	if st.hovered && !st.visible {
		remain := tooltipDelay - c.Now().Sub(st.hoverAt)
		if remain < 0 {
			remain = 0
		}
		c.Animate(remain)
	}
	if st.visible {
		c.Animate(50 * time.Millisecond)
		c.Overlay(buildTooltipOverlay(c, st.anchor, st.text))
	}
}

func buildTooltipOverlay(c *Ctx, anchor render.Rect, text string) *layout.Element {
	th := c.Theme()
	style := th.Typography.Caption
	padX := th.Spacing.S
	padY := th.Spacing.XS
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(text))
	}
	w := tw + 2*padX
	h := lh + 2*padY
	x, y := placeAnchor(anchor, w, h, PlacementBottom)

	host := layout.New(layout.Box().Absolute(x, y).Size(w, h))
	host.Overlay = true
	host.Frame = render.Rect{X: x, Y: y, W: w, H: h}
	msg := text
	host.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		f := host.Frame
		r := th.Radius.Medium
		drawElevationShadow(dl, f, r, th.Elevation.ShadowSm)
		bg := th.Chrome
		dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, bg, th.Border)
		_, mh := eng.MeasureAt(msg, style.Size)
		eng.DrawStringTopAt(dl, msg, f.X+padX, f.Y+(f.H-mh)/2, th.Foreground, style.Size)
	}
	// Tooltips must not steal pointer events from the trigger.
	host.OnMouse = func(_ *layout.Element, _ *input.Mouse) {}
	return host
}

// showTooltipAt registers a one-shot tooltip overlay at anchor (used by Table).
func showTooltipAt(c *Ctx, anchor render.Rect, text string) {
	if text == "" {
		return
	}
	c.Overlay(buildTooltipOverlay(c, anchor, text))
}
