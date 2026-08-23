package uiv2

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

func (app *App) leftNav() ui.View {
	return ui.Nav("app-left-nav", ui.NavVertical, ui.NavIconTop,
		ui.NavItem{ID: "requests", Label: "Requests", Icon: icons.Send},
		ui.NavItem{ID: "environments", Label: "Envs", Icon: icons.FolderPlus},
		ui.NavItem{ID: "spaces", Label: "Spaces", Icon: icons.Boxes},
	).Selected(app.activePageIndex).OnSelectItem(func(i int, _ string) {
		app.activePageIndex = i
	}).Width(75)
}
