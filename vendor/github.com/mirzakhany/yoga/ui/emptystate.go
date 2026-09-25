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
	kids := make([]View, 0, 4)
	if !d.icon.Empty() {
		kids = append(kids, Icon(d.icon, th.Metrics.IconSizeMD*2, th.ForegroundMuted))
	}
	if d.title != "" {
		kids = append(kids, Subtitle(d.title))
	}
	if d.detail != "" {
		kids = append(kids, Muted(d.detail))
	}
	if d.action != nil {
		kids = append(kids, d.action)
	}
	inner := Column(kids...).Gap(th.Spacing.M).Align(AlignCenter)
	return Center(inner).Style(n.spec).Padding(th.Spacing.XXL).Layout(c)
}
