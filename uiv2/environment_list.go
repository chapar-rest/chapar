package uiv2

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

func (app *App) environmentList(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Caption("Environments").PaddingXY(th.Spacing.XS, th.Spacing.XS),
		ui.Row(
			ui.Spacer(),
			ui.Button("btn-import-env", ui.Text("Import")).Secondary().
				IconStart(icons.Download).
				OnClick(func() {}),
			ui.Button("btn-add-env", ui.Text("Add")).Primary().
				IconStart(icons.Plus).
				OnClick(func() {}),
		).Gap(th.Spacing.XS).
			PaddingXY(th.Spacing.XS, th.Spacing.XS),
		ui.TextField("env-list-search", "").
			Placeholder("Search…").
			IconStart(icons.Search).
			OnChange(func(s string) {}).
			MarginLeft(th.Spacing.XS).
			MarginRight(th.Spacing.XS),
		ui.ViewOf(app.envTree).
			Margin(th.Spacing.XS),
	).Grow(1).Background(ui.TokenSurface)
}
