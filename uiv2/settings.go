package uiv2

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

func (app *App) settings(c *ui.Ctx) ui.View {
	return ui.Column(
		ui.Row(
			app.settingsTabs(c),
			app.settingsContent(c),
		).Align(ui.AlignStretch).Grow(1).Background(ui.TokenSurface),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *App) settingsTabs(c *ui.Ctx) ui.View {
	return ui.Nav("settings-nav", ui.NavVertical, ui.NavIconLeft,
		ui.NavItem{ID: "settings-general", Label: "General", Icon: icons.House},
		ui.NavItem{ID: "settings-appearance", Label: "Appearance", Icon: icons.Palette},
		ui.NavItem{ID: "settings-behavior", Label: "Behavior", Icon: icons.Activity},
		ui.NavItem{ID: "settings-advanced", Label: "Advanced", Icon: icons.Pen},
	).Width(200)
}

func (app *App) settingsContent(c *ui.Ctx) ui.View {
	return ui.Column(
		ui.Caption("General").PaddingXY(c.Theme().Spacing.XS, c.Theme().Spacing.XS),
	).Background(ui.TokenSurface)
}

// func (app *App) settingsGeneral(c *ui.Ctx) ui.View {
// 	return ui.Form("form-settings-general",
// 		ui.FormSwitch("f-notify", "HttP version", "Show system alerts and toasts", app.formNotify, func(v bool) {
// 		}),
// 		// ui.FormSelect("f-theme", "Theme", "Application color scheme", themes,
// 		// 	selectIndex(app.formTheme, themeNames()),
// 		// 	func(v string) {
// 		// 		theme.Use(v)
// 		// 		app.formTheme = v
// 		// 		app.theme = v
// 		// 		app.setStatus("theme: " + v)
// 		// 	}),
// 		// ui.FormNumber("f-size", "Font size", "Editor font size in points", app.formSize, 10, 24, 1, func(v float64) {
// 		// 	app.formSize = v
// 		// 	app.setStatus(fmt.Sprintf("font size: %.0f", v))
// 		// }),
// 		// ui.FormText("f-file", "Default file", "Open this file on startup", app.formFile, func(v string) {
// 		// 	app.formFile = v
// 		// 	app.setStatus("default file: " + v)
// 		// }),
// 		// ui.FormSlider("f-vol", "Volume", "Master output level", app.formVol, 0, 100, 1, func(v float64) {
// 		// 	app.formVol = v
// 		// 	app.setStatus(fmt.Sprintf("volume: %.0f", v))
// 		// }),
// 		// ui.FormStepper("f-count", "Retries", "Number of retry attempts", app.formCount, 0, 10, 1, func(v float64) {
// 		// 	app.formCount = v
// 		// 	app.setStatus(fmt.Sprintf("retries: %.0f", v))
// 		// }),
// 	)
// }
