package navigator

import (
	"gioui.org/layout"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/sidebar"
)

type Navigator struct {
	views   map[any]View
	current any

	*sidebar.Sidebar
}

func New() *Navigator {
	return &Navigator{
		views:   make(map[any]View),
		Sidebar: sidebar.New(),
	}
}

func (n *Navigator) Register(view View) {
	detail := view.Info()

	n.views[detail.ID] = view
	if n.current == any(nil) {
		n.current = detail.ID
	}

	n.Sidebar.AddNavItem(sidebar.Item{
		Tag:  detail.ID,
		Name: detail.Title,
		Icon: detail.Icon,
	})
}

func (n *Navigator) SwitchTo(id any) {
	_, ok := n.views[id]
	if !ok {
		return
	}
	n.current = id
}

func (n *Navigator) Current() View {
	return n.views[n.current]
}

func (n *Navigator) Update() {
	if n.Sidebar.Changed() {
		n.SwitchTo(n.Sidebar.Current())
	}
}

func (n *Navigator) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return n.Sidebar.Layout(gtx, th)
}
