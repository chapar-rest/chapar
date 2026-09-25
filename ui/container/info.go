package container

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// InfoPane builds the Info tab for request containers: optional name override and description.
func InfoPane(th *theme.Theme, id string, req *domain.Request, descEd *ui.Editor, markDirty func()) ui.View {
	autoName := domain.RequestAutoName(req)
	return ui.Column(
		ui.Text("Name").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
		ui.TextField("info-name-"+id, domain.RequestInfoNameValue(req)).
			Placeholder(autoName).
			OnChange(func(s string) {
				domain.SetRequestInfoName(req, s)
				markDirty()
			}),
		ui.Text("Description").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)).MarginTop(th.Spacing.S),
		ui.ViewOf(descEd).Grow(1),
	).Gap(th.Spacing.S).Grow(1)
}

// NewDescriptionEditor creates an editor for request metadata.description.
func NewDescriptionEditor(description string) *ui.Editor {
	return ui.NewEditor([]byte(description), highlight.Noop{})
}
