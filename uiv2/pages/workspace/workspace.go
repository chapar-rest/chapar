package workspace

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"

	"github.com/chapar-rest/chapar/uiv2/pages/internal/listui"
	"github.com/chapar-rest/chapar/uiv2/widget"
)

// Section is the workspaces sidebar section.
type Section struct{}

// New constructs the workspaces section.
func New() *Section {
	return &Section{}
}

// MenuItem returns the side menu entry for workspaces.
func (s *Section) MenuItem() widget.SideMenuItem {
	return widget.SideMenuItem{Tag: "workspaces", Name: "Workspaces", Icon: icons.Workspaces}
}

// BuildListPanel renders the workspaces list in the left rail.
func (s *Section) BuildListPanel(panel *core.Frame) {
	listui.SectionHeader(panel, "Workspaces")
	core.NewText(panel).
		SetType(core.TextBodyMedium).
		SetText("Coming soon")
}
