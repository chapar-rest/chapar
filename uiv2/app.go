package uiv2

import (
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/mirzakhany/yoga/ui"
)

type App struct {
	activeEnvironmentId string
	activeWorkspaceId   string

	activePageIndex int

	envTree      *ui.Tree
	requestsTree *ui.Tree

	splitOrientationIcon string

	repo    repository.RepositoryV2
	catalog *Catalog
}

func BuildChaparUI() *App {
	appState := prefs.GetAppState()

	repo, err := repository.NewFilesystemV2(prefs.GetWorkspacePath(), appState.Spec.ActiveWorkspace.Name)
	if err != nil {
		panic(err)
	}

	catalog := newCatalog(repo)
	if err := catalog.Load(); err != nil {
		panic(err)
	}

	envTree := ui.NewTree(nil)
	requestsTree := ui.NewTree(nil)

	app := &App{
		activeEnvironmentId: appState.Spec.SelectedEnvironment.ID,
		activeWorkspaceId:   appState.Spec.ActiveWorkspace.ID,
		activePageIndex:     0,
		envTree:             envTree,
		requestsTree:        requestsTree,
		repo:                repo,
		catalog:             catalog,
	}

	if err := app.loadEnvTree(); err != nil {
		panic(err)
	}
	if err := app.loadRequestsTree(); err != nil {
		panic(err)
	}

	return app
}

func (app *App) Body(c *ui.Ctx) ui.View {
	th := c.Theme()

	var leftPane ui.View
	if app.activePageIndex == 0 {
		leftPane = app.requestsList(c)
	} else if app.activePageIndex == 1 {
		leftPane = app.environmentList(c)
	}

	return ui.Column(
		app.topBar(c),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Row(
			app.leftNav(),
			ui.VLine(th.Stroke.Thin, th.Border),
			ui.Splitter("nav-split", ui.Horizontal, leftPane, ui.Column(ui.Text("Right Pane")).Grow(1).Background(ui.TokenSurface)).Sizes(300, 0),
		).Align(ui.AlignStretch).Grow(1),
		ui.HLine(th.Stroke.Thin, th.Border),
		app.footer(c),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *App) loadEnvTree() error {
	items := make([]*ui.TreeNode, 0, len(app.catalog.Environments))
	for _, env := range app.catalog.Environments {
		items = append(items, &ui.TreeNode{
			Label: env.MetaData.Name,
			Leaf:  true,
		})
	}
	root := &ui.TreeNode{
		Label:    "src",
		Data:     "src",
		Children: items,
	}

	app.envTree.SetRoot(root)
	return nil
}

func (app *App) loadRequestsTree() error {
	items := make([]*ui.TreeNode, 0, len(app.catalog.Collections))
	for _, col := range app.catalog.Collections {
		item := &ui.TreeNode{
			Label:    col.MetaData.Name,
			Children: make([]*ui.TreeNode, 0, len(col.Spec.Requests)),
		}

		for _, req := range col.Spec.Requests {
			item.Children = append(item.Children, &ui.TreeNode{
				Label: req.MetaData.Name,
				Leaf:  true,
			})
		}

		items = append(items, item)
	}

	for _, req := range app.catalog.Requests {
		item := &ui.TreeNode{
			Label: req.MetaData.Name,
			Leaf:  true,
		}
		items = append(items, item)
	}

	root := &ui.TreeNode{
		Label:    "requests",
		Data:     "requests",
		Children: items,
	}

	app.requestsTree.SetRoot(root)
	return nil
}
