package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// CardVariant selects the card's visual elevation level.
type CardVariant int

const (
	CardFlat CardVariant = iota
	CardRaised
	CardElevated
)

type cardData struct {
	title, subtitle string
	body            View
	variant         CardVariant
}

// Card is a surfaced container with optional title, subtitle, and body.
func Card(title, subtitle string, body View) *Node {
	return &Node{kind: kindCard, extra: &cardData{title: title, subtitle: subtitle, body: body, variant: CardRaised}}
}

// Elevated adds a medium drop shadow.
func (n *Node) Elevated() *Node {
	if d, ok := n.extra.(*cardData); ok {
		d.variant = CardElevated
	}
	return n
}

// Flat removes the shadow.
func (n *Node) Flat() *Node {
	if d, ok := n.extra.(*cardData); ok {
		d.variant = CardFlat
	}
	return n
}

func (n *Node) layoutCard(c *Ctx) *layout.Element {
	th := c.Theme()
	d, _ := n.extra.(*cardData)
	if d == nil {
		d = &cardData{variant: CardRaised}
	}
	pad := th.Spacing.L
	var kids []View
	if d.title != "" {
		kids = append(kids, Text(d.title).Size(th.Typography.Subtitle.Size).Weight(th.Typography.Subtitle.Weight))
	}
	if d.subtitle != "" {
		kids = append(kids, Text(d.subtitle).Size(th.Typography.Caption.Size).Style(Spec{}.TextColor(TokenForegroundMuted)))
	}
	if d.body != nil {
		kids = append(kids, d.body)
	}
	el := Column(kids...).Gap(th.Spacing.XS).Padding(pad).Style(n.spec).Layout(c)
	variant := d.variant
	prev := el.Paint
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		r := th.Radius.Large
		switch variant {
		case CardElevated:
			drawElevationShadow(dl, el.Frame, r, th.Elevation.ShadowMd)
		case CardRaised:
			drawElevationShadow(dl, el.Frame, r, th.Elevation.ShadowSm)
		}
		dl.AddRoundedRectBorder(el.Frame, r, th.Stroke.Thin, th.Chrome, th.Border)
		if prev != nil {
			prev(dl, text)
		}
	}
	return el
}
