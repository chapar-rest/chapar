package pages

import (
	"os"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	reqicons "github.com/chapar-rest/chapar/ui/icons"
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
	CloseByID(string)
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
	query    string
	tree     *ui.Tree
	repo     repository.RepositoryV2
	cat      RequestsCatalog
	ws       workspace
	files    func() *ui.FileDialog
	err      func(error)
	SideOpen bool
}

func NewRequestsPage(repo repository.RepositoryV2, cat RequestsCatalog, ws workspace, files func() *ui.FileDialog, errFn func(error)) *Requests {
	p := &Requests{repo: repo, cat: cat, ws: ws, files: files, err: errFn}
	p.tree = ui.NewTree(&ui.TreeNode{Label: "root", Data: "root"})
	p.tree.IconFor = p.iconFor
	p.tree.OnActivate = p.activate
	p.tree.ContextMenu = p.menu
	p.tree.CanDrop = p.canDrop
	p.tree.OnDrop = p.drop
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
	open := p.expandedCollections()
	p.tree.SetRoot(&ui.TreeNode{Label: "root", Data: "root"})
	p.expandCollections(open)
	p.tree.SetFilter(p.query)
}

// expandedCollections records which collections are open. Rebuild replaces the
// root, which would otherwise collapse the whole sidebar under the user.
func (p *Requests) expandedCollections() map[string]bool {
	open := make(map[string]bool)
	root := p.tree.Root()
	if root == nil {
		return open
	}
	for _, n := range root.Children {
		if ref, ok := n.Data.(NodeRef); ok && n.Expanded() {
			open[ref.ID] = true
		}
	}
	return open
}

func (p *Requests) expandCollections(open map[string]bool) {
	root := p.tree.Root()
	if root == nil {
		return
	}
	for _, n := range root.Children {
		if ref, ok := n.Data.(NodeRef); ok && open[ref.ID] {
			p.tree.SetExpanded(n, true)
		}
	}
}

func requestNode(r *domain.Request) *ui.TreeNode {
	icon := reqicons.Badge(r)
	return &ui.TreeNode{
		Label: domain.RequestDisplayName(r),
		Data:  NodeRef{Kind: domain.KindRequest, ID: r.MetaData.ID},
		Leaf:  true,
		Icon:  icon,
	}
}

func (p *Requests) syncTreeLabels() {
	root := p.tree.Root()
	if root == nil {
		return
	}
	var walk func(n *ui.TreeNode)
	walk = func(n *ui.TreeNode) {
		if ref, ok := n.Data.(NodeRef); ok && ref.Kind == domain.KindRequest {
			if req := p.cat.RequestByID(ref.ID); req != nil {
				n.Label = domain.RequestDisplayName(req)
			}
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
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

// dropCollection resolves the collection a drag lands in. A nil collection is
// the top level (standalone requests); ok is false when the drop makes no
// sense, such as a target whose collection no longer exists.
func (p *Requests) dropCollection(ev ui.DropEvent) (col *domain.Collection, ok bool) {
	// A nil target is the empty space below the rows: the top level.
	if ev.Target == nil {
		return nil, true
	}
	ref, isRef := ev.Target.Data.(NodeRef)
	if !isRef {
		return nil, true // the root node
	}
	switch ref.Kind {
	case domain.KindCollection:
		// Dropping onto the row itself moves into the collection; dropping
		// above or below it lands next to it, at the top level.
		if ev.Pos != ui.DropInside {
			return nil, true
		}
		col = p.cat.CollectionByID(ref.ID)
		return col, col != nil
	case domain.KindRequest:
		// Land wherever the request under the cursor lives.
		parentRef, isRef := parentRef(ev.Target)
		if !isRef || parentRef.Kind != domain.KindCollection {
			return nil, true
		}
		col = p.cat.CollectionByID(parentRef.ID)
		return col, col != nil
	default:
		return nil, false
	}
}

func parentRef(n *ui.TreeNode) (NodeRef, bool) {
	parent := n.Parent()
	if parent == nil {
		return NodeRef{}, false
	}
	ref, ok := parent.Data.(NodeRef)
	return ref, ok
}

// canDrop accepts only request moves that change the owning collection.
// Collections themselves are not nestable, so they never move.
func (p *Requests) canDrop(ev ui.DropEvent) bool {
	_, _, ok := p.resolveDrop(ev)
	return ok
}

func (p *Requests) resolveDrop(ev ui.DropEvent) (req *domain.Request, dst *domain.Collection, ok bool) {
	ref, isRef := ev.Source.Data.(NodeRef)
	if !isRef || ref.Kind != domain.KindRequest {
		return nil, nil, false
	}
	req = p.cat.RequestByID(ref.ID)
	if req == nil {
		return nil, nil, false
	}
	dst, ok = p.dropCollection(ev)
	if !ok {
		return nil, nil, false
	}
	dstID := ""
	if dst != nil {
		dstID = dst.MetaData.ID
	}
	if dstID == req.CollectionID {
		return nil, nil, false // already there
	}
	return req, dst, true
}

func (p *Requests) drop(ev ui.DropEvent) {
	req, dst, ok := p.resolveDrop(ev)
	if !ok {
		return
	}
	var src *domain.Collection
	if req.CollectionID != "" {
		src = p.cat.CollectionByID(req.CollectionID)
	}
	if err := p.repo.MoveRequest(req, src, dst); err != nil {
		p.err(err)
		return
	}
	_ = p.cat.Load()
	p.Rebuild()
	if dst != nil {
		// Show the request where it landed.
		p.expandCollections(map[string]bool{dst.MetaData.ID: true})
	}
}

// newMenuItems are the creation entries of the sidebar's New split button. The
// button itself creates an HTTP request; the chevron offers the other kinds.
func (p *Requests) newMenuItems() []ui.MenuItem {
	return []ui.MenuItem{
		{Label: "HTTP request", OnSelect: func() { p.createRequest(domain.RequestTypeHTTP, NodeRef{}) }},
		{Label: "gRPC request", OnSelect: func() { p.createRequest(domain.RequestTypeGRPC, NodeRef{}) }},
		{Label: "GraphQL request", OnSelect: func() { p.createRequest(domain.RequestTypeGraphQL, NodeRef{}) }},
		ui.MenuSeparator,
		{Label: "Collection", OnSelect: p.createCollection},
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
		items = append(items,
			ui.MenuItem{Label: "Duplicate", OnSelect: func() { p.duplicateRef(ref) }},
			ui.MenuItem{Label: "Delete", OnSelect: func() { p.deleteRef(ref) }},
		)
	}
	return items
}

func (p *Requests) duplicateRef(ref NodeRef) {
	switch ref.Kind {
	case domain.KindRequest:
		req := p.cat.RequestByID(ref.ID)
		if req == nil {
			return
		}
		newReq := req.Clone()
		newReq.MetaData.Name += " (copy)"
		var col *domain.Collection
		if req.CollectionID != "" {
			col = p.cat.CollectionByID(req.CollectionID)
		}
		if err := p.repo.CreateRequest(newReq, col); err != nil {
			p.err(err)
			return
		}
		_ = p.cat.Load()
		p.Rebuild()
		p.ws.OpenRequest(newReq)
	case domain.KindCollection:
		col := p.cat.CollectionByID(ref.ID)
		if col == nil {
			return
		}
		colClone := col.Clone()
		if err := p.repo.CreateCollection(colClone); err != nil {
			p.err(err)
			return
		}
		for _, req := range colClone.Spec.Requests {
			if err := p.repo.CreateRequest(req, colClone); err != nil {
				p.err(err)
				return
			}
		}
		_ = p.cat.Load()
		p.Rebuild()
		p.ws.OpenCollection(colClone)
	}
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

// CreateHTTP creates a new HTTP request in the sidebar tree.
func (p *Requests) CreateHTTP() { p.createRequest(domain.RequestTypeHTTP, NodeRef{}) }

// CreateGRPC creates a new gRPC request.
func (p *Requests) CreateGRPC() { p.createRequest(domain.RequestTypeGRPC, NodeRef{}) }

// CreateGraphQL creates a new GraphQL request.
func (p *Requests) CreateGraphQL() { p.createRequest(domain.RequestTypeGraphQL, NodeRef{}) }

// CreateCollection creates a new collection.
func (p *Requests) CreateCollection() { p.createCollection() }

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
		p.ws.CloseByID(ref.ID)
	case domain.KindCollection:
		col := p.cat.CollectionByID(ref.ID)
		if col == nil {
			return
		}
		if err := p.repo.DeleteCollection(col); err != nil {
			p.err(err)
			return
		}
		p.ws.CloseByID(ref.ID)
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
		Title: "Import collection",
		Mode:  ui.FileDialogOpenFile,
		Filters: []ui.FileFilter{
			{Label: "JSON", Exts: []string{".json"}},
			{Label: "YAML", Exts: []string{".yaml", ".yml"}},
			{Label: "Proto", Exts: []string{".proto"}},
		},
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			path := paths[0]
			if strings.HasSuffix(strings.ToLower(path), ".proto") {
				data, err := os.ReadFile(path)
				if err != nil {
					p.err(err)
					return
				}
				if err := importer.ImportProtoFile(data, p.repo, path); err != nil {
					p.err(err)
					return
				}
			} else if err := importer.ImportPostmanCollectionFromFile(path, p.repo); err != nil {
				data, readErr := os.ReadFile(path)
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
	workspace := p.ws.Layout(c)
	if !p.SideOpen {
		return workspace
	}
	return ui.Splitter("req-page-split", ui.Horizontal, p.side(c), workspace).
		Sizes(250, 0).
		HandleOnHover().
		Grow(1)
}

func (p *Requests) side(c *ui.Ctx) ui.View {
	th := c.Theme()
	p.syncTreeLabels()

	return ui.Column(
		ui.Strong("Requests").Margin(th.Spacing.S),
		ui.Row(
			ui.Spacer(),
			ui.Button("req-import", ui.Text("Import")).OnClick(p.importFile),
			ui.MenuButton("req-new", "New", p.newMenuItems()).Primary().IconStart(icons.Plus).
				OnClick(func() { p.createRequest(domain.RequestTypeHTTP, NodeRef{}) }),
		).Gap(th.Spacing.S).MarginRight(th.Spacing.S),
		ui.TextField("req-search", p.query).Placeholder("Search...").IconStart(icons.Search).
			OnChange(func(s string) { p.query = s; p.tree.SetFilter(s) }).Margin(th.Spacing.S),
		ui.ViewOf(p.tree).Grow(1),
	).Gap(th.Spacing.S).Grow(1).Background(ui.TokenChrome)
}
