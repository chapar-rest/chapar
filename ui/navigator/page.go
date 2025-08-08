package navigator

import (
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type PageId string

const (
	RequestsPageId     PageId = "requests"
	EnvironmentsPageId PageId = "environments"
	ProtoFilesPageId   PageId = "protofiles"
	WorkspacesPageId   PageId = "workspaces"
	SettingsPageId     PageId = "settings"
)

type Info struct {
	ID    PageId
	Title string
	Icon  *widget.Icon
}

type View interface {
	Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions
	Info() Info
	OnEnter()
}
