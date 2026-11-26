package container

import (
	"github.com/yogaui/yoga/icons"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/theme"
)

// TabIconer is implemented by containers whose workspace tab shows an icon
// before the title, so requests, collections and environments open side by
// side can be told apart at a glance.
type TabIconer interface {
	TabIcon(th *theme.Theme) (icons.Icon, render.Color)
}

// TabTooltiper is implemented by containers whose workspace tab shows more
// than its title on hover, such as the URL a request is sent to.
type TabTooltiper interface {
	TabTooltip() string
}
