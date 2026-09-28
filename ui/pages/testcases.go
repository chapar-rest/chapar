package pages

import (
	"errors"
	"fmt"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

type testsWorkspace interface {
	OpenTestCase(*domain.TestCase)
	CloseByID(string)
	Layout(*ui.Ctx) ui.View
}

// TestsDeps is what the test cases page works with.
type TestsDeps struct {
	Repo  repository.RepositoryV2
	List  func() []*domain.TestCase
	Get   func(id string) *domain.TestCase
	Load  func() error
	WS    testsWorkspace
	Files func() *ui.FileDialog
	Error func(error)
	// Run runs a test case from the sidebar; nil hides Run.
	Run func(*domain.TestCase)
}

type TestCases struct {
	d        TestsDeps
	query    string
	tree     *ui.Tree
	SideOpen bool
}

func NewTestCasesPage(d TestsDeps) *TestCases {
	p := &TestCases{d: d}
	p.tree = ui.NewTree(&ui.TreeNode{Label: "root"})
	p.tree.OnActivate = p.activate
	p.tree.ContextMenu = p.menu
	p.Rebuild()
	return p
}

func (p *TestCases) Rebuild() {
	cases := p.d.List()
	children := make([]*ui.TreeNode, 0, len(cases))
	for _, tc := range cases {
		children = append(children, &ui.TreeNode{
			Label: tc.MetaData.Name,
			Data:  NodeRef{Kind: domain.KindTestCase, ID: tc.MetaData.ID},
			Leaf:  true,
			Icon:  icons.FlaskConical,
		})
	}
	p.tree.Loader = func(n *ui.TreeNode) []*ui.TreeNode { return children }
	p.tree.SetRoot(&ui.TreeNode{Label: "root"})
	p.tree.SetFilter(p.query)
}

func (p *TestCases) activate(n *ui.TreeNode) {
	ref, ok := n.Data.(NodeRef)
	if !ok {
		return
	}
	if tc := p.d.Get(ref.ID); tc != nil {
		p.d.WS.OpenTestCase(tc)
	}
}

func (p *TestCases) menu(n *ui.TreeNode) []ui.MenuItem {
	ref, _ := n.Data.(NodeRef)
	items := []ui.MenuItem{
		{Label: "New", OnSelect: p.Create},
		{Label: "Import", OnSelect: p.importFile},
	}
	if ref.Kind != domain.KindTestCase {
		return items
	}
	if p.d.Run != nil {
		items = append([]ui.MenuItem{{Label: "Run", OnSelect: func() {
			if tc := p.d.Get(ref.ID); tc != nil {
				p.d.WS.OpenTestCase(tc)
				p.d.Run(tc)
			}
		}}, ui.MenuSeparator}, items...)
	}
	return append(items,
		ui.MenuItem{Label: "Duplicate", OnSelect: func() { p.duplicate(ref.ID) }},
		ui.MenuItem{Label: "Delete", OnSelect: func() { p.delete(ref.ID) }},
	)
}

func (p *TestCases) Create() {
	tc := domain.NewTestCase("New Test Case")
	p.add(tc)
}

func (p *TestCases) add(tc *domain.TestCase) {
	if err := p.d.Repo.CreateTestCase(tc); err != nil {
		p.d.Error(err)
		return
	}
	_ = p.d.Load()
	p.Rebuild()
	p.d.WS.OpenTestCase(tc)
}

func (p *TestCases) duplicate(id string) {
	tc := p.d.Get(id)
	if tc == nil {
		return
	}
	cp := tc.Clone()
	cp.MetaData.Name += " (copy)"
	p.add(cp)
}

func (p *TestCases) delete(id string) {
	tc := p.d.Get(id)
	if tc == nil {
		return
	}
	if err := p.d.Repo.DeleteTestCase(tc); err != nil {
		p.d.Error(err)
		return
	}
	p.d.WS.CloseByID(id)
	_ = p.d.Load()
	p.Rebuild()
}

func (p *TestCases) importFile() {
	if p.d.Files == nil {
		return
	}
	fd := p.d.Files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title:   "Import test case",
		Mode:    ui.FileDialogOpenFile,
		Filters: []ui.FileFilter{{Label: "YAML", Exts: []string{".yaml", ".yml"}}},
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			tc, err := repository.LoadFromYaml[domain.TestCase](paths[0])
			if err != nil {
				p.d.Error(fmt.Errorf("%s is not a test case file: %w", paths[0], err))
				return
			}
			if tc.Kind != domain.KindTestCase {
				p.d.Error(errors.New(paths[0] + " is not a test case file"))
				return
			}
			if tc.MetaData.Name == "" {
				tc.MetaData.Name = "Imported Test Case"
			}
			p.add(tc)
		},
	})
}

func (p *TestCases) Layout(c *ui.Ctx) ui.View {
	workspace := p.d.WS.Layout(c)
	if !p.SideOpen {
		return workspace
	}
	return ui.Splitter("tests-page-split", ui.Horizontal, p.side(c), workspace).
		Sizes(250, 0).
		HandleOnHover().
		Grow(1)
}

func (p *TestCases) side(c *ui.Ctx) ui.View {
	th := c.Theme()
	var list ui.View = ui.ViewOf(p.tree).Grow(1)
	if len(p.d.List()) == 0 {
		list = ui.EmptyState("No test cases", "A test case sends your requests in order and checks their responses.").Grow(1)
	}
	return ui.Column(
		ui.Strong("Test cases").Margin(th.Spacing.S),
		ui.Row(
			ui.Spacer(),
			ui.Button("tests-import", ui.Text("Import")).OnClick(p.importFile),
			ui.Button("tests-new", ui.Text("New")).Primary().IconStart(icons.Plus).OnClick(p.Create),
		).Gap(th.Spacing.S).MarginRight(th.Spacing.S),
		ui.TextField("tests-search", p.query).Placeholder("Search...").IconStart(icons.Search).
			OnChange(func(s string) { p.query = s; p.tree.SetFilter(s) }).Margin(th.Spacing.S),
		list,
	).Gap(th.Spacing.S).Grow(1).Background(ui.TokenChrome)
}
