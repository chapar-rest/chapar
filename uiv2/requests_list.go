package uiv2

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

func (app *App) requestsList(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Caption("Requests").PaddingXY(th.Spacing.XS, th.Spacing.XS),
		ui.Row(
			ui.Spacer(),
			ui.Button("btn-import-req", ui.Text("Import")).Secondary().
				IconStart(icons.Download).
				OnClick(func() {}),
			ui.MenuButton("menu-add-req", "Add", []ui.MenuItem{
				{Label: "Rest API", OnSelect: func() {}},
				{Label: "GraphQL", OnSelect: func() {}},
				{Label: "WebSocket", OnSelect: func() {}},
				{Label: "gRPC", OnSelect: func() {}},
				{Label: "MQTT", OnSelect: func() {}},
			}).Primary().IconStart(icons.Plus),
		).Gap(th.Spacing.XS).
			PaddingXY(th.Spacing.XS, th.Spacing.XS),
		ui.TextField("req-list-search", "").
			Placeholder("Search…").
			IconStart(icons.Search).
			OnChange(func(s string) {}).
			MarginLeft(th.Spacing.XS).
			MarginRight(th.Spacing.XS),
		ui.ViewOf(app.requestsTree).
			MarginTop(th.Spacing.XS).
			MarginLeft(th.Spacing.XS).
			MarginRight(th.Spacing.XS),
	).Grow(1).Background(ui.TokenSurface)
}
