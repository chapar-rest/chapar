package container

import (
	codeeditor "github.com/yogaui/editor"
	"github.com/yogaui/editor/highlight"
	"github.com/yogaui/yoga/theme"
	"github.com/yogaui/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
)

// InfoPane builds the Info tab for request containers: optional name override and description.
func InfoPane(th *theme.Theme, id string, req *domain.Request, descEd *codeeditor.Editor, markDirty func()) ui.View {
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
// It is prose, so it never shows the minimap.
func NewDescriptionEditor(description string) *codeeditor.Editor {
	ed := codeeditor.NewEditor([]byte(description), highlight.Noop{})
	ed.SetMinimap(false)
	return ed
}
