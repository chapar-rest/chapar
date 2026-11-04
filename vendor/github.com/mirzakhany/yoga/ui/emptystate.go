package ui

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
)

type emptyStateData struct {
	title, detail string
	icon          icons.Icon
	action        View
}

// EmptyState is a centered no-data message with optional icon and action.
func EmptyState(title, detail string) *Node {
	return &Node{kind: kindEmptyState, extra: &emptyStateData{title: title, detail: detail}}
}

// EmptyIcon sets the leading icon for EmptyState.
func (n *Node) EmptyIcon(icon icons.Icon) *Node {
	if d, ok := n.extra.(*emptyStateData); ok {
		d.icon = icon
	}
	return n
}

// Action sets the optional action control for EmptyState.
func (n *Node) Action(v View) *Node {
	if d, ok := n.extra.(*emptyStateData); ok {
		d.action = v
	}
	return n
}

func (n *Node) layoutEmptyState(c *Ctx) *layout.Element {
	d, _ := n.extra.(*emptyStateData)
	if d == nil {
		d = &emptyStateData{}
	}
	th := c.Theme()
	center := func(v View) View { return Row(v).Justify(JustifyCenter) }
	kids := make([]View, 0, 4)
	if !d.icon.Empty() {
		kids = append(kids, center(Icon(d.icon, th.Metrics.IconSizeMD*2, th.ForegroundMuted)))
	}
	if d.title != "" {
		kids = append(kids, center(Subtitle(d.title)))
	}
	// The block is as wide as its widest line, and narrower when the room is:
	// the detail wraps then rather than running past the edges.
	var w float32
	if eng := c.Text(); eng != nil {
		if d.title != "" {
			w, _ = eng.MeasureAtWeight(d.title, th.Typography.Subtitle.Size, th.Typography.Subtitle.Weight)
		}
		if d.detail != "" {
			dw, _ := eng.MeasureAt(d.detail, th.Typography.Body.Size)
			w = f32max(w, dw)
		}
	}
	if d.detail != "" {
		kids = append(kids, Paragraph(d.detail).TextAlign(AlignCenter).Style(Spec{}.TextColor(TokenForegroundMuted)))
	}
	if d.action != nil {
		kids = append(kids, center(d.action))
	}
	inner := Column(kids...).Gap(th.Spacing.M).Align(AlignStretch).Shrink(1)
	if w > 0 {
		inner = inner.Width(w + 1)
	}
	return Column(Row(Spacer(), inner, Spacer())).Justify(JustifyCenter).Grow(1).
		Style(n.spec).Padding(th.Spacing.XXL).Layout(c)
}
