package container

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// TabIconer is implemented by containers whose workspace tab shows an icon
// before the title, so requests, collections and environments open side by
// side can be told apart at a glance.
type TabIconer interface {
	TabIcon(th *theme.Theme) (icons.Icon, render.Color)
}
