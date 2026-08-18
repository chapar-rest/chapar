package uiv2

import (
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/chapar-rest/chapar/uiv2/pages"
	"github.com/chapar-rest/chapar/uiv2/sender"
	"github.com/chapar-rest/chapar/uiv2/settings"
	"github.com/chapar-rest/chapar/version"
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const (
	navRequests = iota
	navEnvs
	navProto
	navWorkspaces
)

type App struct {
	repo     repository.RepositoryV2
	catalog  *Catalog
	sender   *sender.Service
	dialogs  *ui.DialogHost
	files    *ui.FileDialog
	toasts   *ui.ToastHost
	settings *settings.Dialog
	confirm  *confirm
	ws       *Workspace

	requests *pages.Requests
	envs     *pages.Environments
	protos   *pages.ProtoFiles
	spaces   *pages.Workspaces

	navIndex int
	search   string
	initErr  error
	wake     func()
	executor scripting.Executor
}

var _ yoga.App = (*App)(nil)
var _ yoga.Closer = (*App)(nil)
var _ yoga.KeyHook = (*App)(nil)

func BuildApp() *App {
	a := &App{
		dialogs: ui.NewDialogHost(),
		files:   ui.NewFileDialog(),
		toasts:  ui.NewToastHost(),
		confirm: &confirm{},
	}
	a.settings = settings.New(func(name string) {
		applyChaparTheme(name)
	}, a.showError, a.toast, nil)

	appState := prefs.GetAppState()
	wsName := domain.DefaultWorkspaceName
	if appState.Spec.ActiveWorkspace != nil && appState.Spec.ActiveWorkspace.Name != "" {
		wsName = appState.Spec.ActiveWorkspace.Name
	}
	repo, err := repository.NewFilesystemV2(prefs.GetWorkspacePath(), wsName)
	if err != nil {
		a.initErr = err
		return a
	}
	a.repo = repo
	a.catalog = newCatalog(repo)
	if err := a.catalog.Load(); err != nil {
		a.initErr = err
		return a
	}

	a.sender = sender.New(repo, a.catalog.RequestByID, a.catalog.CollectionByID, a.catalog.ProtoFileList, func(env *domain.Environment) {
		a.catalog.ReplaceEnvironment(env)
	})

	a.ws = newWorkspace(a.deps, a.confirm)
	a.ws.onTrees = a.rebuildTrees

	a.requests = pages.NewRequestsPage(repo, a.catalog, a.ws, a.files, a.showError)
	a.envs = pages.NewEnvironmentsPage(repo,
		func() []*domain.Environment { return a.catalog.Environments },
		a.catalog.EnvironmentByID,
		a.catalog.Load,
		a.ws, a.files, a.showError)
	a.protos = pages.NewProtoFilesPage(repo, a.catalog.ProtoFileList, a.catalog.Load, a.files, a.showError)
	a.spaces = pages.NewWorkspacesPage(repo,
		func() []*domain.Workspace { return a.catalog.Workspaces },
		a.catalog.Load, a.showError, a.switchWorkspace)

	go a.initScripting()
	return a
}

func (a *App) deps() container.Deps {
	return container.Deps{
		Repo:      a.repo,
		Catalog:   a.catalog,
		Sender:    a.sender,
		Dialogs:   a.dialogs,
		Files:     a.files,
		Toasts:    a.toasts,
		Wake:      a.wake,
		ActiveEnv: a.catalog.ActiveEnvironment,
		Report: container.Reporter{
			Error: a.showError,
		},
	}
}

func (a *App) rebuildTrees() {
	if a.requests != nil {
		a.requests.Rebuild()
	}
	if a.envs != nil {
		a.envs.Rebuild()
	}
	if a.protos != nil {
		a.protos.Reload()
	}
	if a.spaces != nil {
		a.spaces.Reload()
	}
}

func (a *App) showError(err error) {
	if err == nil {
		return
	}
	if a.dialogs != nil {
		a.dialogs.ShowError("Error", err.Error(), nil)
	}
}

func (a *App) toast(msg string) {
	if a.toasts != nil {
		a.toasts.Show(msg, ui.ToastInfo, 3*time.Second)
	}
}

func (a *App) switchWorkspace(ws *domain.Workspace) {
	if a.ws.HasDirty() {
		a.confirm.open = true
		a.confirm.message = "Switching workspace will close unsaved tabs. Continue?"
		a.confirm.onYes = func() { a.doSwitchWorkspace(ws) }
		return
	}
	a.doSwitchWorkspace(ws)
}

func (a *App) doSwitchWorkspace(ws *domain.Workspace) {
	a.ws.CloseAll()
	if err := a.catalog.SetActiveWorkspace(ws); err != nil {
		a.showError(err)
		return
	}
	if err := a.catalog.Load(); err != nil {
		a.showError(err)
		return
	}
	a.rebuildTrees()
}

func (a *App) initScripting() {
	cfg := prefs.GetGlobalConfig().Spec.Scripting
	if !cfg.Enabled {
		return
	}
	exec, err := scripting.GetExecutor(cfg.Language, cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("scripting: %v", err))
		return
	}
	if err := exec.Init(cfg); err != nil {
		logger.Error(fmt.Sprintf("scripting init: %v", err))
		return
	}
	a.executor = exec
}

func (a *App) Body(c *ui.Ctx) ui.View {
	a.wake = c.Invalidate
	th := c.Theme()

	if a.initErr != nil {
		return ui.Column(
			ui.Title("Chapar failed to start"),
			ui.Text(a.initErr.Error()),
			a.dialogs,
		).Padding(th.Spacing.L).Grow(1).Background(ui.TokenSurface)
	}

	page := a.pageView(c)
	shell := ui.Column(
		a.topBar(c),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Row(
			a.nav(c),
			ui.VLine(th.Stroke.Thin, th.Border),
			ui.ViewOf(page).Grow(1),
		).Align(ui.AlignStretch).Grow(1),
		ui.HLine(th.Stroke.Thin, th.Border),
		a.footer(c),
		a.dialogs,
		a.files,
		a.toasts,
	).Grow(1).Background(ui.TokenSurface)

	overlays := []ui.View{shell}
	if a.settings != nil && a.settings.Open {
		overlays = append(overlays, a.settings.Layout(c))
	}
	if a.confirm != nil && a.confirm.open {
		overlays = append(overlays, a.confirmLayout(c))
	}
	if len(overlays) == 1 {
		return shell
	}
	return ui.Stack(overlays...).Grow(1)
}

func (a *App) pageView(c *ui.Ctx) ui.View {
	switch a.navIndex {
	case navEnvs:
		return a.envs.Layout(c)
	case navProto:
		return a.protos.Layout(c)
	case navWorkspaces:
		return a.spaces.Layout(c)
	default:
		return a.requests.Layout(c)
	}
}

func (a *App) topBar(c *ui.Ctx) ui.View {
	th := c.Theme()
	wsOpts := []ui.SelectOption{}
	wsSel := 0
	for i, w := range a.catalog.Workspaces {
		wsOpts = append(wsOpts, ui.SelectOption{Label: w.MetaData.Name, Value: w.MetaData.ID})
		if w.MetaData.ID == a.catalog.ActiveWorkspaceID {
			wsSel = i
		}
	}
	if len(wsOpts) == 0 {
		wsOpts = []ui.SelectOption{{Label: "Default Workspace", Value: "default"}}
	}

	envOpts := []ui.SelectOption{{Label: "No Environment", Value: ""}}
	envSel := 0
	for i, e := range a.catalog.Environments {
		envOpts = append(envOpts, ui.SelectOption{Label: e.MetaData.Name, Value: e.MetaData.ID})
		if e.MetaData.ID == a.catalog.ActiveEnvID {
			envSel = i + 1
		}
	}

	hits := a.searchHits()
	var hitViews []ui.View
	for i, h := range hits {
		if i > 7 {
			break
		}
		h := h
		hitViews = append(hitViews, ui.Button("hit-"+h.id, ui.Text(h.title)).Subtle().OnClick(func() { a.openHit(h) }))
	}

	row := ui.Row(
		ui.Select("top-ws", wsOpts).Width(180).Selected(wsSel).OnChange(func(v string) {
			if ws := a.catalog.WorkspaceByID(v); ws != nil {
				a.switchWorkspace(ws)
			}
		}),
		ui.TextField("top-search", a.search).Placeholder("Search…").IconStart("search").Grow(1).
			OnChange(func(s string) { a.search = s }).
			OnSubmit(func(s string) {
				if hs := a.searchHits(); len(hs) > 0 {
					a.openHit(hs[0])
				}
			}),
		ui.Select("top-env", envOpts).Width(180).Selected(envSel).OnChange(func(v string) {
			_ = a.catalog.SetActiveEnv(v)
		}),
		ui.IconButton("top-settings", "settings").OnClick(func() { a.settings.Show() }),
	).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.S)

	if len(hitViews) == 0 || a.search == "" {
		return row
	}
	return ui.Column(row, ui.Row(hitViews...).Gap(th.Spacing.XS).PaddingXY(th.Spacing.M, 0))
}

type searchHit struct {
	id, kind, title string
}

func (a *App) searchHits() []searchHit {
	q := strings.ToLower(strings.TrimSpace(a.search))
	if q == "" {
		return nil
	}
	var out []searchHit
	match := func(id, kind, title string) {
		if strings.Contains(strings.ToLower(title), q) {
			out = append(out, searchHit{id: id, kind: kind, title: title})
		}
	}
	for _, e := range a.catalog.Environments {
		match(e.MetaData.ID, domain.KindEnv, e.MetaData.Name)
	}
	for _, col := range a.catalog.Collections {
		match(col.MetaData.ID, domain.KindCollection, col.MetaData.Name)
		for _, r := range col.Spec.Requests {
			match(r.MetaData.ID, domain.KindRequest, r.MetaData.Name)
		}
	}
	for _, r := range a.catalog.Requests {
		match(r.MetaData.ID, domain.KindRequest, r.MetaData.Name)
	}
	return out
}

func (a *App) openHit(h searchHit) {
	switch h.kind {
	case domain.KindEnv:
		if env := a.catalog.EnvironmentByID(h.id); env != nil {
			a.navIndex = navEnvs
			a.ws.OpenEnv(env)
		}
	case domain.KindCollection:
		if col := a.catalog.CollectionByID(h.id); col != nil {
			a.navIndex = navRequests
			a.ws.OpenCollection(col)
		}
	case domain.KindRequest:
		if req := a.catalog.RequestByID(h.id); req != nil {
			a.navIndex = navRequests
			a.ws.OpenRequest(req)
		}
	}
	a.search = ""
}

func (a *App) nav(c *ui.Ctx) ui.View {
	return ui.Nav("main-nav", ui.NavVertical, ui.NavIconTop,
		ui.NavItem{ID: "requests", Label: "Requests", Icon: "folder"},
		ui.NavItem{ID: "environments", Label: "Envs", Icon: "list"},
		ui.NavItem{ID: "protofiles", Label: "Protos", Icon: "code"},
		ui.NavItem{ID: "workspaces", Label: "Spaces", Icon: "grid"},
	).Selected(a.navIndex).OnSelectItem(func(i int, _ string) { a.navIndex = i }).Width(88)
}

func (a *App) footer(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Row(
		ui.Caption("Chapar "+version.GetAppVersion()),
		ui.Spacer(),
		ui.Caption("Yoga UI"),
	).PaddingXY(th.Spacing.M, th.Spacing.XS)
}

func (a *App) confirmLayout(c *ui.Ctx) ui.View {
	th := c.Theme()
	panel := ui.Column(
		ui.Title("Confirm"),
		ui.Text(a.confirm.message),
		ui.Row(
			ui.Spacer(),
			ui.Button("confirm-no", ui.Text("Cancel")).OnClick(func() { a.confirm.open = false }),
			ui.Button("confirm-yes", ui.Text("Close")).Primary().OnClick(func() {
				fn := a.confirm.onYes
				a.confirm.open = false
				if fn != nil {
					fn()
				}
			}),
		).Gap(th.Spacing.S),
	).Gap(th.Spacing.M).Padding(th.Spacing.L).Width(420).Background(ui.TokenChrome)
	return ui.Center(panel).Grow(1).BackgroundColor(render.RGBA8(0, 0, 0, 140))
}

func (a *App) ClearColor() render.Color { return theme.Current().Background }

func (a *App) Close() {
	if a.ws != nil {
		a.ws.CloseAll()
	}
	if a.executor != nil {
		_ = a.executor.Shutdown()
	}
}

func (a *App) OnKey(_ *ui.Ctx, k input.KeyEvent) bool {
	if !k.Mods.Primary() {
		return false
	}
	switch k.Key {
	case input.KeyS:
		a.ws.SaveActive()
		return true
	case input.KeyEnter:
		a.ws.SendActive()
		return true
	}
	return false
}
