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
				c.MarkNeedsPaint()
			}
			st.anchor = e.Frame
			if !st.visible && now.Sub(st.hoverAt) >= tooltipDelay {
				st.visible = true
				c.MarkNeedsPaint()
			}
		} else if st.hovered || st.visible {
			st.hovered = false
			st.visible = false
			c.MarkNeedsPaint()
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

// hoverCardMaxWidth bounds a hover card, so a long value wraps to the next
// line instead of running off the window.
const hoverCardMaxWidth = 360

// buildHoverCardOverlay builds the two-line card a field shows for the part of
// its value under the pointer: a title in the accent text color and the value
// beneath it, wrapped to a few lines.
func buildHoverCardOverlay(c *Ctx, anchor render.Rect, card HoverCard) *layout.Element {
	th := c.Theme()
	title, body := th.Typography.Caption, th.Typography.Body
	padX, padY := th.Spacing.S, th.Spacing.XS
	eng := c.Text()

	maxText := float32(hoverCardMaxWidth) - 2*padX
	lines := wrapToWidth(eng, card.Body, body.Size, maxText)
	var w, lineH float32
	if eng != nil {
		tw, th2 := eng.MeasureAt(card.Title, title.Size)
		w, lineH = tw, th2
		for _, ln := range lines {
			if lw, lh := eng.MeasureAt(ln, body.Size); lw > w {
				w, lineH = lw, f32max(lineH, lh)
			} else if lh > lineH {
				lineH = lh
			}
		}
	} else {
		lineH = body.LineHeight
		w = maxText
	}
	rows := float32(len(lines))
	if card.Title != "" {
		rows++
	}
	boxW := f32min(w+2*padX, hoverCardMaxWidth)
	boxH := rows*lineH + 2*padY
	x, y := placeAnchor(anchor, boxW, boxH, PlacementBottom)

	host := layout.New(layout.Box().Absolute(x, y).Size(boxW, boxH))
	host.Overlay = true
	host.Frame = render.Rect{X: x, Y: y, W: boxW, H: boxH}
	host.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		f := host.Frame
		r := th.Radius.Medium
		drawElevationShadow(dl, f, r, th.Elevation.ShadowSm)
		dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, th.Chrome, th.Border)
		dl.PushClip(f)
		ty := f.Y + padY
		if card.Title != "" {
			eng.DrawStringTopAt(dl, card.Title, f.X+padX, ty, th.Info, title.Size)
			ty += lineH
		}
		for _, ln := range lines {
			eng.DrawStringTopAt(dl, ln, f.X+padX, ty, th.Foreground, body.Size)
			ty += lineH
		}
		dl.PopClip()
	}
	// The card must not steal pointer events from the field under it.
	host.OnMouse = func(_ *layout.Element, _ *input.Mouse) {}
	return host
}

// hoverCardMaxLines bounds how much of a long value a card shows.
const hoverCardMaxLines = 4

// wrapToWidth breaks s into at most hoverCardMaxLines lines that fit maxW,
// ending the last one in an ellipsis when the text runs past that.
func wrapToWidth(eng *shape.Engine, s string, size, maxW float32) []string {
	if s == "" {
		return nil
	}
	if eng == nil || maxW <= 0 {
		return []string{s}
	}
	var lines []string
	rest := []rune(s)
	for len(rest) > 0 {
		n := len(rest)
		for n > 1 {
			if w, _ := eng.MeasureAt(string(rest[:n]), size); w <= maxW {
				break
			}
			n--
		}
		if len(lines) == hoverCardMaxLines-1 && n < len(rest) {
			// Last line: keep room for the ellipsis rather than cutting mid-word.
			for n > 1 {
				if w, _ := eng.MeasureAt(string(rest[:n])+"…", size); w <= maxW {
					break
				}
				n--
			}
			lines = append(lines, string(rest[:n])+"…")
			break
		}
		lines = append(lines, string(rest[:n]))
		rest = rest[n:]
	}
	return lines
}
