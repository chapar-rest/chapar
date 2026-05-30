package pages

import (
	"cogentcore.org/core/core"

	"github.com/chapar-rest/chapar/uiv2/widget"
)

// Section is a navigable sidebar section with its own list panel.
type Section interface {
	MenuItem() widget.SideMenuItem
	BuildListPanel(panel *core.Frame)
}

// FullPageSection is a section that renders in the main content area instead
// of using the list panel and tab view.
type FullPageSection interface {
	Section
	BuildContent(content *core.Frame)
}
