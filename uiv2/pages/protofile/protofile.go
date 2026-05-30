package protofile

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"

	"github.com/chapar-rest/chapar/uiv2/pages/internal/listui"
	"github.com/chapar-rest/chapar/uiv2/widget"
)

// Section is the proto files sidebar section.
type Section struct{}

// New constructs the proto files section.
func New() *Section {
	return &Section{}
}

// MenuItem returns the side menu entry for proto files.
func (s *Section) MenuItem() widget.SideMenuItem {
	return widget.SideMenuItem{Tag: "protofiles", Name: "Proto", Icon: icons.Folder}
}

// BuildListPanel renders the proto files list in the left rail.
func (s *Section) BuildListPanel(panel *core.Frame) {
	listui.SectionHeader(panel, "Proto Files")
	core.NewText(panel).
		SetType(core.TextBodyMedium).
		SetText("Coming soon")
}
