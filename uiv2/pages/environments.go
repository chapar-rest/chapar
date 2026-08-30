package pages

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

type Environments struct {
	query string
	tree  *ui.Tree
	repo  repository.RepositoryV2
	list  func() []*domain.Environment
	get   func(id string) *domain.Environment
	load  func() error
	ws    workspace
	files func() *ui.FileDialog
	err   func(error)
}

func NewEnvironmentsPage(repo repository.RepositoryV2, list func() []*domain.Environment, get func(id string) *domain.Environment, load func() error, ws workspace, files func() *ui.FileDialog, errFn func(error)) *Environments {
	p := &Environments{repo: repo, list: list, get: get, load: load, ws: ws, files: files, err: errFn}
	p.tree = ui.NewTree(&ui.TreeNode{Label: "root"})
	p.tree.OnActivate = p.activate
	p.tree.ContextMenu = p.menu
	p.Rebuild()
	return p
}

func (p *Environments) Rebuild() {
	envs := p.list()
	children := make([]*ui.TreeNode, 0, len(envs))
	for _, e := range envs {
		children = append(children, &ui.TreeNode{
			Label: e.MetaData.Name,
			Data:  NodeRef{Kind: domain.KindEnv, ID: e.MetaData.ID},
			Leaf:  true,
			Icon:  icons.FolderPlus,
		})
	}
	p.tree.Loader = func(n *ui.TreeNode) []*ui.TreeNode { return children }
	p.tree.SetRoot(&ui.TreeNode{Label: "root"})
	p.tree.SetFilter(p.query)
}

func (p *Environments) activate(n *ui.TreeNode) {
	ref, ok := n.Data.(NodeRef)
	if !ok {
		return
	}
	if env := p.get(ref.ID); env != nil {
		p.ws.OpenEnv(env)
	}
}

func (p *Environments) menu(n *ui.TreeNode) []ui.MenuItem {
	ref, _ := n.Data.(NodeRef)
	items := []ui.MenuItem{
		{Label: "New", OnSelect: p.create},
		{Label: "Import", OnSelect: p.importFile},
	}
	if ref.Kind == domain.KindEnv {
		items = append(items,
			ui.MenuItem{Label: "Duplicate", OnSelect: func() { p.duplicateID(ref.ID) }},
			ui.MenuItem{Label: "Delete", OnSelect: func() { p.deleteID(ref.ID) }},
		)
	}
	return items
}

func (p *Environments) duplicateID(id string) {
	env := p.get(id)
	if env == nil {
		return
	}
	newEnv := env.Clone()
	newEnv.MetaData.Name += " (copy)"
	if err := p.repo.CreateEnvironment(newEnv); err != nil {
		p.err(err)
		return
	}
	_ = p.load()
	p.Rebuild()
	p.ws.OpenEnv(newEnv)
}

func (p *Environments) Create() { p.create() }

func (p *Environments) create() {
	env := domain.NewEnvironment("New Environment")
	if err := p.repo.CreateEnvironment(env); err != nil {
		p.err(err)
		return
	}
	_ = p.load()
	p.Rebuild()
	p.ws.OpenEnv(env)
}

func (p *Environments) deleteID(id string) {
	env := p.get(id)
	if env == nil {
		return
	}
	if err := p.repo.DeleteEnvironment(env); err != nil {
		p.err(err)
		return
	}
	p.ws.CloseByID(id)
	_ = p.load()
	p.Rebuild()
}

func (p *Environments) importFile() {
	if p.files == nil {
		return
	}
	fd := p.files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title:   "Import environment",
		Mode:    ui.FileDialogOpenFile,
		Filters: []ui.FileFilter{{Label: "JSON", Exts: []string{".json"}}},
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			if err := importer.ImportPostmanEnvironmentFromFile(paths[0], p.repo); err != nil {
				p.err(err)
				return
			}
			_ = p.load()
			p.Rebuild()
		},
	})
}

func (p *Environments) Layout(c *ui.Ctx) ui.View {
	return ui.Splitter("env-page-split", ui.Horizontal, p.side(c), p.ws.Layout(c)).Sizes(280, 0).Grow(1)
}

func (p *Environments) side(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Strong("Environments").Margin(th.Spacing.S),
		ui.Row(
			ui.Spacer(),
			ui.Button("env-import", ui.Text("Import")).OnClick(p.importFile),
			ui.Button("env-new", ui.Text("New")).Primary().IconStart(icons.Plus).OnClick(p.create),
		).Gap(th.Spacing.S).MarginRight(th.Spacing.S),
		ui.TextField("env-search", p.query).Placeholder("Search...").IconStart(icons.Search).
			OnChange(func(s string) { p.query = s; p.tree.SetFilter(s) }).Margin(th.Spacing.S),
		ui.ViewOf(p.tree).Grow(1),
	).Gap(th.Spacing.S).Grow(1)
}
