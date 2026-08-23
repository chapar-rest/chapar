package uiv2

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

func (app *App) footer(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Row(
		ui.Caption("v0.3.1").MarginLeft(th.Spacing.S),
		ui.Spacer(),
		ui.Button("btn-oriantation", nil).
			IconStart(icons.SquareSplitHorizontal).
			Ghost().
			HoverFill().
			OnClick(func() {
				if app.splitOrientationIcon == "split_vertical" {
					app.splitOrientationIcon = "split_horizontal"
				} else {
					app.splitOrientationIcon = "split_vertical"
				}
			}),
		ui.Button("btn-console", ui.Text("Console")).IconStart(icons.Terminal).Ghost().HoverFill().OnClick(func() {
		}),
		ui.Button("btn-notifications", ui.Text("Notifications")).
			IconStart(icons.Bell).
			Ghost().
			HoverFill().
			MarginRight(th.Spacing.S).
			OnClick(func() {
			}),
	)
}
