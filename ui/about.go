package ui

import (
	"fmt"

	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/ui/langsrv"
	"github.com/chapar-rest/chapar/version"
)

const websiteURL = "https://chapar.rest"

// openAbout shows what Chapar is and where to read more about it.
func (a *App) openAbout(c *ui.Ctx) {
	c.Dialogs().Show(ui.DialogOpts{
		Title:  "About Chapar",
		Width:  420,
		Height: 220,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.Paragraph("Chapar is an open-source, native API testing tool for HTTP, gRPC and GraphQL requests."),
				ui.Paragraph("Version "+version.GetAppVersion()).
					Size(th.Typography.Caption.Size).
					Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
				ui.Link("about-website", websiteURL).OnClick(func() {
					if err := langsrv.OpenURL(websiteURL); err != nil {
						logger.Error(fmt.Sprintf("opening %s: %v", websiteURL, err))
					}
				}),
			).Gap(th.Spacing.M).Padding(th.Spacing.M)
		},
		Actions: []ui.DialogAction{
			{Label: "Close", Primary: true},
		},
	})
}
