package pages

import (
	"os"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	reqicons "github.com/chapar-rest/chapar/uiv2/icons"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type NodeRef struct {
	Kind string
	ID   string
}

type workspace interface {
	OpenRequest(*domain.Request)
	OpenCollection(*domain.Collection)
	OpenEnv(*domain.Environment)
	Layout(*ui.Ctx) ui.View
}

type RequestsCatalog interface {
	AllCollections() []*domain.Collection
	StandaloneRequests() []*domain.Request
	Load() error
	RequestByID(id string) *domain.Request
	CollectionByID(id string) *domain.Collection
}

type Requests struct {
	query string
	tree  *ui.Tree
	repo  repository.RepositoryV2
	cat   RequestsCatalog
	ws    workspace
	files func() *ui.FileDialog
	err   func(error)
}

func NewRequestsPage(repo repository.RepositoryV2, cat RequestsCatalog, ws workspace, files func() *ui.FileDialog, errFn func(error)) *Requests {
	p := &Requests{repo: repo, cat: cat, ws: ws, files: files, err: errFn}
	p.tree = ui.NewTree(&ui.TreeNode{Label: "root", Data: "root"})
	p.tree.Background = &theme.Current().Panel
	p.tree.IconFor = p.iconFor
	p.tree.OnActivate = p.activate
	p.tree.ContextMenu = p.menu
	p.Rebuild()
	return p
}

func (p *Requests) Rebuild() {
	cols := p.cat.AllCollections()
	reqs := p.cat.StandaloneRequests()
	children := make([]*ui.TreeNode, 0, len(cols)+len(reqs))
	for _, col := range cols {
		children = append(children, &ui.TreeNode{
			Label: col.MetaData.Name,
			Data:  NodeRef{Kind: domain.KindCollection, ID: col.MetaData.ID},
			Icon:  icons.Folder,
		})
	}
	for _, r := range reqs {
		children = append(children, requestNode(r))
	}
	p.tree.Loader = func(n *ui.TreeNode) []*ui.TreeNode {
		ref, ok := n.Data.(NodeRef)
		if !ok {
			return children
		}
		if ref.Kind != domain.KindCollection {
			return nil
		}
		col := p.cat.CollectionByID(ref.ID)
		if col == nil {
			return nil
		}
		out := make([]*ui.TreeNode, 0, len(col.Spec.Requests))
		for _, r := range col.Spec.Requests {
			out = append(out, requestNode(r))
		}
		return out
	}
	p.tree.SetRoot(&ui.TreeNode{Label: "root", Data: "root"})
	p.tree.SetFilter(p.query)
}

const requestBadgeIconSize = float32(22)

func requestNode(r *domain.Request) *ui.TreeNode {
	icon := reqicons.Badge(r)
	return &ui.TreeNode{
		Label: r.MetaData.Name,
		Data:  NodeRef{Kind: domain.KindRequest, ID: r.MetaData.ID},
		Leaf:  true,
		Icon:  icon,
	}
}

func (p *Requests) iconFor(n *ui.TreeNode, expanded bool) (icons.Icon, render.Color) {
	th := theme.Current()
	ref, ok := n.Data.(NodeRef)
	if !ok {
		return icons.File, th.ForegroundMuted
	}
	switch ref.Kind {
	case domain.KindCollection:
		name := n.ClosedIcon
		if name.Empty() {
			name = icons.Folder
		}
		if expanded {
			if !n.OpenIcon.Empty() {
				name = n.OpenIcon
			} else {
				name = icons.FolderOpen
			}
		}
		return name, th.Accent
	case domain.KindRequest:
		if req := p.cat.RequestByID(ref.ID); req != nil {
			return reqicons.Badge(req), reqicons.Color(req, th)
		}
		if !n.Icon.Empty() {
			return n.Icon, th.ForegroundMuted
		}
		return reqicons.Badge(nil), reqicons.Color(nil, th)
	default:
		if !n.Icon.Empty() {
			return n.Icon, th.ForegroundMuted
		}
		return icons.File, th.ForegroundMuted
	}
}

func (p *Requests) activate(n *ui.TreeNode) {
	ref, ok := n.Data.(NodeRef)
	if !ok {
		return
	}
	switch ref.Kind {
	case domain.KindRequest:
		if req := p.cat.RequestByID(ref.ID); req != nil {
			p.ws.OpenRequest(req)
		}
	case domain.KindCollection:
		if col := p.cat.CollectionByID(ref.ID); col != nil {
			p.ws.OpenCollection(col)
		}
	}
}

func (p *Requests) menu(n *ui.TreeNode) []ui.MenuItem {
	ref, _ := n.Data.(NodeRef)
	items := []ui.MenuItem{
		{Label: "New HTTP request", OnSelect: func() { p.createRequest(domain.RequestTypeHTTP, ref) }},
		{Label: "New gRPC request", OnSelect: func() { p.createRequest(domain.RequestTypeGRPC, ref) }},
		{Label: "New GraphQL request", OnSelect: func() { p.createRequest(domain.RequestTypeGraphQL, ref) }},
		{Label: "New collection", OnSelect: p.createCollection},
		{Label: "Import", OnSelect: p.importFile},
	}
	if ref.Kind == domain.KindCollection {
		items = append([]ui.MenuItem{{Label: "Open", OnSelect: func() { p.activate(n) }}}, items...)
	}
	if ref.Kind == domain.KindRequest || ref.Kind == domain.KindCollection {
		items = append(items, ui.MenuItem{Label: "Delete", OnSelect: func() { p.deleteRef(ref) }})
	}
	return items
}

func (p *Requests) createRequest(kind domain.RequestType, ref NodeRef) {
	var req *domain.Request
	switch kind {
	case domain.RequestTypeGRPC:
		req = domain.NewGRPCRequest("New Request")
	case domain.RequestTypeGraphQL:
		req = domain.NewGraphQLRequest("New Request")
	default:
		req = domain.NewHTTPRequest("New Request")
	}
	var col *domain.Collection
	if ref.Kind == domain.KindCollection {
		col = p.cat.CollectionByID(ref.ID)
		if col != nil {
			req.CollectionID = col.MetaData.ID
			req.CollectionName = col.MetaData.Name
		}
	}
	if err := p.repo.CreateRequest(req, col); err != nil {
		p.err(err)
		return
	}
	_ = p.cat.Load()
	p.Rebuild()
	p.ws.OpenRequest(req)
}

func (p *Requests) createCollection() {
	col := domain.NewCollection("New Collection")
	if err := p.repo.CreateCollection(col); err != nil {
		p.err(err)
		return
	}
	_ = p.cat.Load()
	p.Rebuild()
	p.ws.OpenCollection(col)
}

func (p *Requests) deleteRef(ref NodeRef) {
	switch ref.Kind {
	case domain.KindRequest:
		req := p.cat.RequestByID(ref.ID)
		if req == nil {
			return
		}
		var col *domain.Collection
		if req.CollectionID != "" {
			col = p.cat.CollectionByID(req.CollectionID)
		}
		if err := p.repo.DeleteRequest(req, col); err != nil {
			p.err(err)
			return
		}
	case domain.KindCollection:
		col := p.cat.CollectionByID(ref.ID)
		if col == nil {
			return
		}
		if err := p.repo.DeleteCollection(col); err != nil {
			p.err(err)
			return
		}
	}
	_ = p.cat.Load()
	p.Rebuild()
}

func (p *Requests) importFile() {
	if p.files == nil {
		return
	}
	fd := p.files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title:   "Import collection",
		Mode:    ui.FileDialogOpenFile,
		Filters: []ui.FileFilter{{Label: "JSON", Exts: []string{".json"}}},
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			if err := importer.ImportPostmanCollectionFromFile(paths[0], p.repo); err != nil {
				data, readErr := os.ReadFile(paths[0])
				if readErr != nil {
					p.err(err)
					return
				}
				if err2 := importer.ImportOpenAPISpec(data, p.repo); err2 != nil {
					p.err(err)
					return
				}
			}
			_ = p.cat.Load()
			p.Rebuild()
		},
	})
}

func (p *Requests) Layout(c *ui.Ctx) ui.View {
	return ui.Splitter("req-page-split", ui.Horizontal, p.side(c), p.ws.Layout(c)).Sizes(280, 0).Grow(1)
}

func (p *Requests) side(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Strong("Requests").Margin(th.Spacing.S),
		ui.Row(
			ui.Spacer(),
			ui.Button("req-import", ui.Text("Import")).OnClick(p.importFile),
			ui.Button("req-new", ui.Text("New")).Primary().IconStart(icons.Plus).OnClick(func() {
				p.createRequest(domain.RequestTypeHTTP, NodeRef{})
			}),
		).Gap(th.Spacing.S).MarginRight(th.Spacing.S),
		ui.TextField("req-search", p.query).Placeholder("Search...").IconStart(icons.Search).
			OnChange(func(s string) { p.query = s; p.tree.SetFilter(s) }).Margin(th.Spacing.S),
		ui.ViewOf(p.tree).Grow(1),
	).Gap(th.Spacing.S).Background(ui.TokenChrome).Grow(1)
}
