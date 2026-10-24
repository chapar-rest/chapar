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
	tone   BadgeTone
	hue    Hue
	hasHue bool
	mono   bool
	tag    bool // Tag: small radius instead of a pill
}

// Hue names a categorical color (see theme.Theme.Categorical). Unlike a
// BadgeTone it carries no status meaning: use it to tell apart HTTP methods,
// environments or tags.
type Hue = theme.Hue

const (
	HueRed    = theme.HueRed
	HueOrange = theme.HueOrange
	HueYellow = theme.HueYellow
	HueGreen  = theme.HueGreen
	HueTeal   = theme.HueTeal
	HueBlue   = theme.HueBlue
	HuePurple = theme.HuePurple
	HuePink   = theme.HuePink
)

// Badge is a compact status/count pill.
func Badge(label string) *Node {
	return &Node{kind: kindBadge, text: label, extra: &badgeData{tone: BadgeMuted}}
}

// Tag is a small label chip, like Badge but with squared-off corners, for
// tags and environment names. Color it with Hue or Tone.
func Tag(label string) *Node {
	return &Node{kind: kindBadge, text: label, extra: &badgeData{tone: BadgeMuted, tag: true}}
}

// badgeStyle is the coloring of a Badge, Tag or RangeBar, or nil.
func (n *Node) badgeStyle() *badgeData {
	switch d := n.extra.(type) {
	case *badgeData:
		return d
	case *rangeData:
		return &d.style
	}
	return nil
}

// Tone sets Badge, Tag or RangeBar coloring.
func (n *Node) Tone(t BadgeTone) *Node {
	if d := n.badgeStyle(); d != nil {
		d.tone = t
		d.hasHue = false
	}
	return n
}

// Hue colors a Badge or Tag with a categorical hue: the hue's text on its
// tinted fill, readable at caption size in every theme. On a RangeBar it
// colors the segment.
func (n *Node) Hue(h Hue) *Node {
	if d := n.badgeStyle(); d != nil && h >= 0 && h < theme.NumHues {
		d.hue, d.hasHue = h, true
	}
	return n
}

// Mono draws a Badge or Tag label in the monospace font, so short codes
// such as HTTP methods line up.
func (n *Node) Mono() *Node {
	if d, ok := n.extra.(*badgeData); ok {
		d.mono = true
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
	mono := d != nil && d.mono
	measure := func(eng *shape.Engine, s string) (float32, float32) {
		if mono {
			return eng.MeasureMonoAt(s, style.Size)
		}
		return eng.MeasureAt(s, style.Size)
	}
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = measure(eng, n.text)
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
	if d != nil && d.hasHue {
		bg, fg = th.CategoricalSurface[d.hue], th.Categorical[d.hue]
	}
	radius := th.Radius.Circular
	if d != nil && d.tag {
		radius = th.Radius.Medium
	}
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		dl.AddRoundedRect(f, radius, bg)
		_, mh := measure(text, label)
		if mono {
			text.DrawStringTopMonoAt(dl, label, f.X+padX, f.Y+(f.H-mh)/2, fg, style.Size)
			return
		}
		text.DrawStringTopAt(dl, label, f.X+padX, f.Y+(f.H-mh)/2, fg, style.Size)
	}
	return el
}

// badgeColors returns the tinted fill and the label color for a tone. Both
// come from the palette: tinting the status hue at a fixed alpha and then
// drawing the same hue on top leaves the label at the hue's own contrast,
// which amber cannot meet on a light surface.
func badgeColors(th *theme.Theme, tone BadgeTone) (bg, fg render.Color) {
	switch tone {
	case BadgeAccent:
		return th.InfoSurface, th.InfoForeground
	case BadgeSuccess:
		return th.SuccessSurface, th.SuccessForeground
	case BadgeWarning:
		return th.WarningSurface, th.WarningForeground
	case BadgeError:
		return th.ErrorSurface, th.ErrorForeground
	default:
		return th.ChromeMuted, th.ForegroundMuted
	}
}
