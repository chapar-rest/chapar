package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// Kbd renders a keyboard-shortcut keycap chip.
func Kbd(label string) *Node {
	return &Node{kind: kindKbd, text: label}
}

func (n *Node) layoutKbd(c *Ctx) *layout.Element {
	th := c.Theme()
	style := th.Typography.Caption
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	padX := th.Spacing.SNudge
	padY := th.Spacing.XXS
	w := tw + 2*padX
	h := lh + 2*padY
	el := layout.New(applyLayoutSpec(layout.Box().Size(w, h).FlexShrink(0), n.spec))
	label := n.text
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		bg := th.ChromeMuted
		dl.AddRoundedRectBorder(f, th.Radius.Small, th.Stroke.Thin, bg, th.Border)
		_, mh := text.MeasureAt(label, style.Size)
		fg := th.ForegroundMuted
		text.DrawStringTopAt(dl, label, f.X+padX, f.Y+(f.H-mh)/2, fg, style.Size)
	}
	return el
}
