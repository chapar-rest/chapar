package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Text renders a string. .Size(n) sets the font size in logical pixels.
// .Weight(w) selects Regular (400) or SemiBold (600+).
// Color and size inherit from the parent environment (e.g. a Button's TextColor)
// unless overridden with Style.
func Text(s string) *Node {
	return &Node{kind: kindText, text: s}
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
	st := applyLayoutSpec(layout.Box().Size(tw+padL+padR, lh+padT+padB).FlexShrink(0), n.spec)
	el := layout.New(st)
	content := n.text
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		pad := el.Style.Padding
		x := el.Frame.X + pad.Left
		contentH := el.Frame.H - pad.Top - pad.Bottom
		y := el.Frame.Y + pad.Top
		if contentH > lh {
			y += (contentH - lh) / 2
		}
		text.DrawStringTopAtWeight(dl, content, x, y, col, size, weight)
	}
	_ = render.Color{}
	return el
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
