package container

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// ActionsNav indices for ActionsPane.
const (
	ActionsBefore  = 0
	ActionsAfter   = 1
	ActionsExtract = 2
)

// ActionsOpts configures the Actions request tab (Before / After / Extract).
type ActionsOpts struct {
	ID string

	Selected *int

	Deps Deps

	Pre        *domain.PreRequest
	Post       *domain.PostRequest
	PreOpts    PrePostOpts
	PostOpts   PrePostOpts
	PreScript  **ui.Editor
	PostScript **ui.Editor
	Preview    string

	// Vars is the response-extractor table. Nil omits the Extract nav item (GraphQL).
	Vars             *ui.Table
	DefaultVarStatus int

	MarkDirty func()
}

// ActionsPane renders a top horizontal nav with Before / After / Extract content.
func ActionsPane(th *theme.Theme, opts ActionsOpts) ui.View {
	if opts.Selected == nil {
		return ui.Column()
	}
	items := []ui.NavItem{
		{ID: "before", Label: "Before"},
		{ID: "after", Label: "After"},
	}
	if opts.Vars != nil {
		items = append(items, ui.NavItem{ID: "extract", Label: "Extract"})
	}
	if *opts.Selected < 0 || *opts.Selected >= len(items) {
		*opts.Selected = ActionsBefore
	}

	var content ui.View
	switch *opts.Selected {
	case ActionsAfter:
		content = PostRequestPane(th, opts.Deps, opts.Post, opts.PostOpts, opts.PostScript, opts.Preview, opts.MarkDirty)
	case ActionsExtract:
		if opts.Vars != nil {
			content = ui.Column(
				ui.Row(
					ui.Button("actions-var-add-"+opts.ID, ui.Text("Add")).OnClick(func() {
						AddVariableRow(opts.Vars, opts.DefaultVarStatus, opts.MarkDirty)
					}),
				).PaddingXY(0, th.Spacing.XS),
				ui.ViewOf(opts.Vars).Grow(1),
			).Grow(1)
		}
	default:
		content = PreRequestPane(th, opts.Deps, opts.Pre, opts.PreOpts, opts.PreScript, opts.MarkDirty)
	}
	if content == nil {
		content = ui.Column()
	}

	return ui.Column(
		ui.Nav("actions-nav-"+opts.ID, ui.NavHorizontal, ui.NavIconLeft, items...).
			Selected(*opts.Selected).
			OnSelectItem(func(i int, _ string) { *opts.Selected = i }).
			Radius(th.Radius.Small),
		ui.Column(content).PaddingXY(0, th.Spacing.S).Gap(th.Spacing.S).Grow(1),
	).Grow(1)
}
