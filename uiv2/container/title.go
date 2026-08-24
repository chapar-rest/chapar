package container

import (
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// TitleRow builds the breadcrumb title row used by request containers:
// [PREFIX] [Collection /] [EditableLabel name] … actions (typically Save).
func TitleRow(th *theme.Theme, id, prefix, collection, name string, prefixColor render.Color, onSave func(string), actions ...ui.View) ui.View {
	breadcrumb := []ui.View{
		ui.Strong(strings.ToUpper(prefix)).
			Style(ui.Spec{}.TextColorLit(prefixColor)),
	}
	if collection != "" {
		breadcrumb = append(breadcrumb,
			ui.Text(collection+" /").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
		)
	}
	breadcrumb = append(breadcrumb,
		ui.EditableLabel("title-"+id, name).
			Placeholder("Name").
			OnSave(onSave),
	)

	right := make([]ui.View, 0, len(actions)+1)
	right = append(right, actions...)
	return ui.Row(
		ui.Row(breadcrumb...).Gap(th.Spacing.XS).Align(ui.AlignCenter),
		ui.Spacer(),
		ui.Row(right...).Gap(th.Spacing.S).Align(ui.AlignCenter),
	).Justify(ui.JustifyBetween).
		PaddingTop(th.Spacing.XS).
		PaddingBottom(th.Spacing.XS).
		PaddingLeft(th.Spacing.M).
		PaddingRight(th.Spacing.M)
}

// SimpleTitleRow is the title row for collection and environment containers:
// [EditableLabel name] … actions.
func SimpleTitleRow(th *theme.Theme, id, name string, onSave func(string), actions ...ui.View) ui.View {
	return ui.Row(
		ui.EditableLabel("title-"+id, name).
			Placeholder("Name").
			OnSave(onSave).
			Grow(1),
		ui.Row(actions...).Gap(th.Spacing.S).Align(ui.AlignCenter),
	).Gap(th.Spacing.S).
		Justify(ui.JustifyBetween).
		Align(ui.AlignCenter).
		PaddingTop(th.Spacing.XS).
		PaddingBottom(th.Spacing.S).
		PaddingLeft(th.Spacing.M).
		PaddingRight(th.Spacing.M)
}

// RenameRequest persists a request name immediately on EditableLabel commit.
func RenameRequest(deps Deps, req *domain.Request, name string) {
	name = strings.TrimSpace(name)
	if name == "" || name == req.MetaData.Name {
		return
	}
	req.MetaData.Name = name
	deps.ReportTitle(name)
	var col *domain.Collection
	if req.CollectionID != "" && deps.Catalog != nil {
		col = deps.Catalog.CollectionByID(req.CollectionID)
	}
	if err := deps.Repo.UpdateRequest(req, col); err != nil {
		deps.ShowError(err)
		return
	}
	if deps.Report.Saved != nil {
		deps.Report.Saved()
	}
}

// RenameCollection persists a collection name immediately on EditableLabel commit.
func RenameCollection(deps Deps, col *domain.Collection, name string) {
	name = strings.TrimSpace(name)
	if name == "" || name == col.MetaData.Name {
		return
	}
	col.MetaData.Name = name
	deps.ReportTitle(name)
	if err := deps.Repo.UpdateCollection(col); err != nil {
		deps.ShowError(err)
		return
	}
	if deps.Report.Saved != nil {
		deps.Report.Saved()
	}
}

// RenameEnvironment persists an environment name immediately on EditableLabel commit.
func RenameEnvironment(deps Deps, env *domain.Environment, name string) {
	name = strings.TrimSpace(name)
	if name == "" || name == env.MetaData.Name {
		return
	}
	env.MetaData.Name = name
	deps.ReportTitle(name)
	if err := deps.Repo.UpdateEnvironment(env); err != nil {
		deps.ShowError(err)
		return
	}
	if deps.Catalog != nil {
		deps.Catalog.ReplaceEnvironment(env)
	}
	if deps.Report.Saved != nil {
		deps.Report.Saved()
	}
}
