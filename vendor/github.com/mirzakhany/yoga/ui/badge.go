package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// BadgeTone selects badge coloring.
type BadgeTone int

const (
	BadgeMuted BadgeTone = iota
	BadgeAccent
	BadgeSuccess
	BadgeWarning
	BadgeError
)

type badgeData struct {
	tone BadgeTone
}

// Badge is a compact status/count pill.
func Badge(label string) *Node {
	return &Node{kind: kindBadge, text: label, extra: &badgeData{tone: BadgeMuted}}
}

// Tone sets Badge coloring.
func (n *Node) Tone(t BadgeTone) *Node {
	if d, ok := n.extra.(*badgeData); ok {
		d.tone = t
	}
	return n
}

func (n *Node) layoutBadge(c *Ctx) *layout.Element {
	th := c.Theme()
	d, _ := n.extra.(*badgeData)
	tone := BadgeMuted
	if d != nil {
		tone = d.tone
	}
	style := th.Typography.Caption
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	padX := th.Spacing.S
	padY := th.Spacing.XXS
	w := tw + 2*padX
	h := lh + 2*padY
	el := layout.New(applyLayoutSpec(layout.Box().Size(w, h).FlexShrink(0), n.spec))
	label := n.text
	bg, fg := badgeColors(th, tone)
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		dl.AddRoundedRect(f, th.Radius.Circular, bg)
		_, mh := text.MeasureAt(label, style.Size)
		text.DrawStringTopAt(dl, label, f.X+padX, f.Y+(f.H-mh)/2, fg, style.Size)
	}
	return el
}

func badgeColors(th *theme.Theme, tone BadgeTone) (bg, fg render.Color) {
	switch tone {
	case BadgeAccent:
		bg = th.Accent
		bg.A = 0.22
		return bg, th.Accent
	case BadgeSuccess:
		bg = th.Success
		bg.A = 0.22
		return bg, th.Success
	case BadgeWarning:
		bg = th.Warning
		bg.A = 0.22
		return bg, th.Warning
	case BadgeError:
		bg = th.Error
		bg.A = 0.22
		return bg, th.Error
	default:
		return th.ChromeMuted, th.ForegroundMuted
	}
}
