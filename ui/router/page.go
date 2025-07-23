package router

import (
	"gioui.org/layout"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/sidebar"
)

type Page interface {
	Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions
	SideBarItem() sidebar.Item
}
