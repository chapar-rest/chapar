package pages

import (
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/mirzakhany/yoga/ui"
)

type Workspaces struct {
	query string
	table *ui.Table
	repo  repository.RepositoryV2
	list  func() []*domain.Workspace
	load  func() error
	err   func(error)
	onUse func(*domain.Workspace)
}

func NewWorkspacesPage(repo repository.RepositoryV2, list func() []*domain.Workspace, load func() error, errFn func(error), onUse func(*domain.Workspace)) *Workspaces {
	p := &Workspaces{repo: repo, list: list, load: load, err: errFn, onUse: onUse}
	p.table = ui.NewTable([]ui.TableColumn{
		{ID: "name", Label: "Name", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: "delete", Tooltip: "Delete"}})
	p.table.Actions[0].OnClick = func(rowID string) { p.deleteID(rowID) }
	p.table.OnCellChange = func(rowID, colID, value string) {
		if colID != "name" {
			return
		}
		for _, w := range p.list() {
			if w.MetaData.ID == rowID {
				w.MetaData.Name = value
				if err := p.repo.UpdateWorkspace(w); err != nil {
					p.err(err)
				}
				_ = p.load()
				return
			}
		}
	}
	p.table.OnRowActivate = func(rowID string) {
		for _, w := range p.list() {
			if w.MetaData.ID == rowID && p.onUse != nil {
				p.onUse(w)
			}
		}
	}
	p.Reload()
	return p
}

func (p *Workspaces) Reload() {
	items := p.list()
	rows := make([]ui.TableRow, 0, len(items))
	q := strings.ToLower(p.query)
	for _, w := range items {
		if q != "" && !strings.Contains(strings.ToLower(w.MetaData.Name), q) {
			continue
		}
		rows = append(rows, ui.TableRow{
			ID:    w.MetaData.ID,
			Icon:  "folder",
			Cells: map[string]string{"name": w.MetaData.Name},
		})
	}
	p.table.SetRows(rows)
}

func (p *Workspaces) deleteID(id string) {
	for _, w := range p.list() {
		if w.MetaData.ID == id {
			if w.MetaData.ID == "default" || w.MetaData.Name == domain.DefaultWorkspaceName {
				return
			}
			if err := p.repo.DeleteWorkspace(w); err != nil {
				p.err(err)
				return
			}
			break
		}
	}
	_ = p.load()
	p.Reload()
}

func (p *Workspaces) create() {
	ws := domain.NewWorkspace("New Workspace")
	if err := p.repo.CreateWorkspace(ws); err != nil {
		p.err(err)
		return
	}
	_ = p.load()
	p.Reload()
}

func (p *Workspaces) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Row(
			ui.Strong("Workspaces"),
			ui.Spacer(),
			ui.TextField("ws-search", p.query).Placeholder("Search...").IconStart("search").Width(220).
				OnChange(func(s string) { p.query = s; p.Reload() }),
			ui.Button("ws-new", ui.Text("New")).Primary().IconStart("add").OnClick(p.create),
		).Gap(th.Spacing.S).Padding(th.Spacing.M),
		ui.Muted("Double-click a workspace to switch.").PaddingXY(th.Spacing.M, 0),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(p.table).Grow(1),
	).Grow(1)
}
