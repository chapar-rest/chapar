package uiv2

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/uiv2/settings"
	"github.com/chapar-rest/chapar/uiv2/theme"
)

func Run() {
	theme.Init()
	settings.Init()
	settings.LoadOrLog()

	b := core.NewBody("Chapar")
	NewAppBar(b)

	// Lay out the body as a row: side menu on the left, content on the right.
	b.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 1)
	})

	menu := NewSideMenu(b)
	menu.AddItem(SideMenuItem{Tag: "requests", Name: "Requests", Icon: icons.SwapHoriz})
	menu.AddItem(SideMenuItem{Tag: "environments", Name: "Envs", Icon: icons.Menu})
	menu.AddItem(SideMenuItem{Tag: "protofiles", Name: "Proto", Icon: icons.Folder})
	menu.AddItem(SideMenuItem{Tag: "workspaces", Name: "Workspaces", Icon: icons.Workspaces})
	menu.AddItem(SideMenuItem{Tag: "settings", Name: "Settings", Icon: icons.Settings})

	content := core.NewFrame(b)
	content.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Padding.Set(units.Dp(16))
	})
	core.NewText(content).
		SetType(core.TextHeadlineSmall).
		SetText("Requests")

	menu.OnSelect(func(item SideMenuItem) {
		content.DeleteChildren()
		core.NewText(content).
			SetType(core.TextHeadlineSmall).
			SetText(item.Name)
		content.Update()
	})

	b.NewWindow().SetDisplayTitle(false).RunMain()
}
