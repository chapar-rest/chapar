package ui

import (
	"github.com/mirzakhany/yoga/input"
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
	actions         []View
	footer          View
	selected        bool
}

type cardState struct {
	hovered, pressed bool
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

// Actions puts views, such as an IconButton or a "..." MenuButton, at the
// right end of the card's title row. Their clicks do not reach the card.
func (n *Node) Actions(views ...View) *Node {
	if d, ok := n.extra.(*cardData); ok {
		d.actions = views
	}
	return n
}

// Footer puts a view below the card's body.
func (n *Node) Footer(v View) *Node {
	if d, ok := n.extra.(*cardData); ok {
		d.footer = v
	}
	return n
}

// CardSelected paints the card's border in the accent color.
func (n *Node) CardSelected(v bool) *Node {
	if d, ok := n.extra.(*cardData); ok {
		d.selected = v
	}
	return n
}

// ClickableCard is a Card that highlights under the pointer and runs the
// handler set with OnClick when clicked. id keys hover and press across
// frames.
func ClickableCard(id, title, subtitle string, body View) *Node {
	n := Card(title, subtitle, body)
	n.id = id
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
	var head []View
	if d.title != "" {
		head = append(head, Text(d.title).Size(th.Typography.Subtitle.Size).Weight(th.Typography.Subtitle.Weight).Ellipsis(EllipsisEnd).Grow(1))
	}
	if len(d.actions) > 0 {
		if d.title == "" {
			head = append(head, Spacer())
		}
		head = append(head, d.actions...)
	}
	switch {
	case len(d.actions) > 0:
		kids = append(kids, Row(head...).Gap(th.Spacing.XS).Align(AlignCenter))
	case len(head) > 0:
		kids = append(kids, head[0])
	}
	if d.subtitle != "" {
		kids = append(kids, Text(d.subtitle).Size(th.Typography.Caption.Size).Style(Spec{}.TextColor(TokenForegroundMuted)))
	}
	if d.body != nil {
		kids = append(kids, d.body)
	}
	if d.footer != nil {
		kids = append(kids, d.footer)
	}
	el := Column(kids...).Gap(th.Spacing.XS).Padding(pad).Style(n.spec).Layout(c)
	var st *cardState
	if n.id != "" {
		st = c.Widget(n.id, func() any { return &cardState{} }).(*cardState)
	}
	variant := d.variant
	selected := d.selected
	prev := el.Paint
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		r := th.Radius.Large
		switch variant {
		case CardElevated:
			drawElevationShadow(dl, el.Frame, r, th.Elevation.ShadowMd)
		case CardRaised:
			drawElevationShadow(dl, el.Frame, r, th.Elevation.ShadowSm)
		}
		fill, border := th.Chrome, th.Border
		if st != nil && st.hovered {
			fill = th.ListHover
		}
		if selected {
			border = th.Accent
		}
		dl.AddRoundedRectBorder(el.Frame, r, th.Stroke.Thin, fill, border)
		if prev != nil {
			prev(dl, text)
		}
	}
	if st != nil {
		onClick := n.onClick
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			inside := e.Frame.Contains(m.X, m.Y)
			trackHover(c, &st.hovered, inside)
			if inside && onClick != nil {
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
	}
	return el
}
