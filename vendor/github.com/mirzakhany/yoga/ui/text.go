package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// EllipsisMode selects how Text shortens a string that does not fit its box.
type EllipsisMode int

const (
	// EllipsisNone keeps Text at its measured width; it never shrinks.
	EllipsisNone EllipsisMode = iota
	// EllipsisEnd keeps the head of the string: "a/very/long/pa…".
	EllipsisEnd
	// EllipsisMiddle keeps the head and the tail, which is what long file
	// paths need: "/Users/me/…/api/v1.proto".
	EllipsisMiddle
)

const ellipsisRune = "…"

// Text renders a string. .Size(n) sets the font size in logical pixels.
// .Weight(w) selects Regular (400) or SemiBold (600+).
// Color and size inherit from the parent environment (e.g. a Button's TextColor)
// unless overridden with Style.
func Text(s string) *Node {
	return &Node{kind: kindText, text: s}
}

// Ellipsis lets Text shrink below its measured width and paint a shortened
// string ending in "…" instead of overflowing its parent. The node keeps its
// full width as the flex basis, so it only shortens when the parent is too
// narrow; give it Grow or a MaxWidth to control how much room it claims.
func (n *Node) Ellipsis(m EllipsisMode) *Node {
	if n.kind == kindText {
		n.ellipsis = m
	}
	return n
}

func (n *Node) layoutText(c *Ctx) *layout.Element {
	th := c.Theme()
	size := c.env.fontSize
	if size <= 0 {
		size = th.Typography.Body.Size
	}
	if n.spec.hasFontSize {
		size = n.spec.fontSize
	}
	weight := shape.WeightRegular
	if n.spec.hasFontWeight {
		weight = n.spec.fontWeight
	}
	col := c.env.textColor
	if !c.env.hasColor {
		col = th.Foreground
	}
	r := n.spec.resolve(th, interactState{})
	if r.hasFg {
		col = r.fg
	}
	if r.hasFontSize {
		size = r.fontSize
	}
	if r.hasFontWeight {
		weight = r.fontWeight
	}

	eng := c.Text()
	var tw, lh float32
	if eng != nil {
		tw, lh = eng.MeasureAtWeight(n.text, size, weight)
	} else {
		tw, lh = size*0.5*float32(len(n.text)), size
	}
	// Width/Height are border-box. Include padding so glyphs keep their
	// measured size inside the content box; paint insets by Style.Padding.
	var padL, padR, padT, padB float32
	if n.spec.hasPad {
		padL, padR = n.spec.pad.Left, n.spec.pad.Right
		padT, padB = n.spec.pad.Top, n.spec.pad.Bottom
	}
	box := layout.Box().Size(tw+padL+padR, lh+padT+padB).FlexShrink(0)
	mode := n.ellipsis
	if mode != EllipsisNone {
		// Shrinking is what makes the ellipsis reachable: keep the measured
		// width as the basis but let the parent take it down to nothing.
		box = box.FlexShrink(1).Min(0, box.MinHeight)
	}
	st := applyLayoutSpec(box, n.spec)
	el := layout.New(st)
	content := n.text
	// Truncation is resolved at paint time, when the final frame is known.
	// Both fields are only ever touched from the paint pass.
	lastAvail := float32(-1)
	shown := content
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		pad := el.Style.Padding
		x := el.Frame.X + pad.Left
		contentH := el.Frame.H - pad.Top - pad.Bottom
		y := el.Frame.Y + pad.Top
		if contentH > lh {
			y += (contentH - lh) / 2
		}
		draw := content
		if mode != EllipsisNone {
			avail := el.Frame.W - pad.Left - pad.Right
			if avail != lastAvail {
				lastAvail = avail
				shown = truncateToWidth(text, content, size, weight, avail, mode)
			}
			draw = shown
		}
		text.DrawStringTopAtWeight(dl, draw, x, y, col, size, weight)
	}
	_ = render.Color{}
	return el
}

// truncateToWidth shortens s so that it fits avail logical pixels, ending (or
// hinging) on "…". It returns s unchanged when it already fits.
func truncateToWidth(eng *shape.Engine, s string, size float32, weight int, avail float32, mode EllipsisMode) string {
	if eng == nil || mode == EllipsisNone || s == "" {
		return s
	}
	width := func(v string) float32 {
		w, _ := eng.MeasureAtWeight(v, size, weight)
		return w
	}
	if avail <= 0 {
		return ""
	}
	if width(s) <= avail {
		return s
	}
	if width(ellipsisRune) > avail {
		return ""
	}
	// Width grows with the number of kept runes, so the largest count that
	// still fits can be found by bisection. len(runes) itself is excluded:
	// the full string is already known not to fit.
	runes := []rune(s)
	lo, hi := 0, len(runes)-1
	best := ellipsisRune
	for lo <= hi {
		mid := (lo + hi) / 2
		cand := elide(runes, mid, mode)
		if width(cand) <= avail {
			best = cand
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return best
}

// elide keeps n of the given runes and stands the rest down to "…", at the end
// or in the middle.
func elide(runes []rune, n int, mode EllipsisMode) string {
	if n <= 0 {
		return ellipsisRune
	}
	if n >= len(runes) {
		return string(runes)
	}
	if mode == EllipsisMiddle {
		head := (n + 1) / 2
		tail := n - head
		return string(runes[:head]) + ellipsisRune + string(runes[len(runes)-tail:])
	}
	return string(runes[:n]) + ellipsisRune
}

// Title is title-ramp text in SemiBold.
func Title(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Title.Size).Weight(th.Typography.Title.Weight)
}

// Subtitle is semibold subtitle text.
func Subtitle(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Subtitle.Size).Weight(th.Typography.Subtitle.Weight)
}

// Caption is small muted text.
func Caption(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Caption.Size).Style(Spec{}.TextColor(TokenForegroundMuted))
}

// Strong is semibold body text.
func Strong(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.BodyStrong.Size).Weight(th.Typography.BodyStrong.Weight)
}

// Muted is body text in ForegroundMuted.
func Muted(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Body.Size).Style(Spec{}.TextColor(TokenForegroundMuted))
}
