package uiv2

import (
	"fmt"
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
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
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
	settings *settings.Panel
	ws       *Workspace

	requests *pages.Requests
	envs     *pages.Environments
	protos   *pages.ProtoFiles
	spaces   *pages.Workspaces

	navIndex int
	initErr  error
	wake     func()
	uiCtx    *ui.Ctx
	executor scripting.Executor
}

var _ yoga.App = (*App)(nil)
var _ yoga.Closer = (*App)(nil)
var _ yoga.KeyHook = (*App)(nil)

func BuildApp() *App {
	a := &App{}
	a.settings = settings.New(func(name string) {
		applyChaparTheme(name)
	})

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

	a.ws = newWorkspace(a.deps, a.confirmClose)
	a.ws.onTrees = a.rebuildTrees
	a.ws.onSettings = func() {
		if a.uiCtx != nil {
			a.openSettings(a.uiCtx)
		}
	}

	files := a.files
	a.requests = pages.NewRequestsPage(repo, a.catalog, a.ws, files, a.showError)
	a.envs = pages.NewEnvironmentsPage(repo,
		func() []*domain.Environment { return a.catalog.Environments },
		a.catalog.EnvironmentByID,
		a.catalog.Load,
		a.ws, files, a.showError)
	a.protos = pages.NewProtoFilesPage(repo, a.catalog.ProtoFileList, a.catalog.Load, files, a.showError)
	a.spaces = pages.NewWorkspacesPage(repo,
		func() []*domain.Workspace { return a.catalog.Workspaces },
		a.catalog.Load, a.showError, a.switchWorkspace)

	go a.initScripting()
	return a
}

func (a *App) files() *ui.FileDialog {
	if a.uiCtx == nil {
		return nil
	}
	return a.uiCtx.Files()
}

func (a *App) dialogs() *ui.DialogHost {
	if a.uiCtx == nil {
		return nil
	}
	return a.uiCtx.Dialogs()
}

func (a *App) toasts() *ui.ToastHost {
	if a.uiCtx == nil {
		return nil
	}
	return a.uiCtx.Toasts()
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
			Toast: a.toast,
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
	if host := a.dialogs(); host != nil {
		host.ShowError("Error", err.Error(), nil)
	}
}

func (a *App) toast(msg string) {
	if host := a.toasts(); host != nil {
		host.Show(msg, ui.ToastInfo, 3*time.Second)
	}
}

func (a *App) confirmClose(title, message string, onYes func()) {
	if host := a.dialogs(); host != nil {
		host.ShowAction(title, message, onYes, nil)
		return
	}
	if onYes != nil {
		onYes()
	}
}

func (a *App) switchWorkspace(ws *domain.Workspace) {
	if a.ws.HasDirty() {
		a.confirmClose("Switch workspace?", "Switching workspace will close unsaved tabs. Continue?", func() {
			a.doSwitchWorkspace(ws)
		})
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
	a.uiCtx = c
	a.wake = c.Invalidate
	th := c.Theme()

	if a.initErr != nil {
		return ui.Column(
			ui.Title("Chapar failed to start"),
			ui.Text(a.initErr.Error()),
		).Padding(th.Spacing.L).Grow(1).Background(ui.TokenSurface)
	}

	a.registerCommands(c)

	return ui.Column(
		a.topBar(c),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Row(
			a.nav(c),
			ui.VLine(th.Stroke.Thin, th.Border),
			ui.ViewOf(a.pageView(c)).Grow(1),
		).Align(ui.AlignStretch).Grow(1),
		ui.HLine(th.Stroke.Thin, th.Border),
		a.footer(c),
	).Grow(1).Background(ui.TokenSurface)
}

func (a *App) registerCommands(c *ui.Ctx) {
	cmds := []*ui.Command{
		ui.Section("Navigation"),
		ui.Cmd("nav.requests").Title("Go to Requests").Icon(icons.Send).Run(func() { a.navIndex = navRequests }),
		ui.Cmd("nav.envs").Title("Go to Environments").Icon(icons.FolderPlus).Run(func() { a.navIndex = navEnvs }),
		ui.Cmd("nav.protos").Title("Go to Proto files").Icon(icons.Code).Run(func() { a.navIndex = navProto }),
		ui.Cmd("nav.spaces").Title("Go to Workspaces").Icon(icons.Boxes).Run(func() { a.navIndex = navWorkspaces }),
		ui.Cmd("app.settings").Title("Open Settings").Shortcut("⌘,").Icon(icons.Settings).Run(func() { a.openSettings(c) }),
		ui.Cmd("file.save").Title("Save").Shortcut("⌘S").Icon(icons.Save).Run(func() { a.ws.SaveActive() }),
		ui.Cmd("file.send").Title("Send / Invoke").Shortcut("⌘Enter").Icon(icons.Play).Run(func() { a.ws.SendActive() }),
		ui.Section("Open"),
	}
	for _, e := range a.catalog.Environments {
		e := e
		cmds = append(cmds, ui.Item("open.env."+e.MetaData.ID).
			Title(e.MetaData.Name).
			Detail("Environment").
			Icon(icons.FolderPlus).
			Run(func() {
				a.navIndex = navEnvs
				a.ws.OpenEnv(e)
			}))
	}
	for _, col := range a.catalog.Collections {
		col := col
		cmds = append(cmds, ui.Item("open.col."+col.MetaData.ID).
			Title(col.MetaData.Name).
			Detail("Collection").
			Icon(icons.Folder).
			Run(func() {
				a.navIndex = navRequests
				a.ws.OpenCollection(col)
			}))
		for _, r := range col.Spec.Requests {
			r := r
			cmds = append(cmds, ui.Item("open.req."+r.MetaData.ID).
				Title(r.MetaData.Name).
				Detail(col.MetaData.Name).
				Icon(icons.File).
				Run(func() {
					a.navIndex = navRequests
					a.ws.OpenRequest(r)
				}))
		}
	}
	for _, r := range a.catalog.Requests {
		r := r
		cmds = append(cmds, ui.Item("open.req."+r.MetaData.ID).
			Title(r.MetaData.Name).
			Detail("Request").
			Icon(icons.File).
			Run(func() {
				a.navIndex = navRequests
				a.ws.OpenRequest(r)
			}))
	}
	c.Commands().Register(cmds...)
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

	return ui.TitleBar(
		ui.Select("top-ws", wsOpts).Width(180).Selected(wsSel).OnChange(func(v string) {
			if ws := a.catalog.WorkspaceByID(v); ws != nil {
				a.switchWorkspace(ws)
			}
		}),
		ui.IconButton("top-add", icons.Plus).OnClick(func() {}),
		ui.Spacer(),
		ui.Button("cmd-palette", ui.Text("Commands")).Width(300).
			IconStart(icons.Search).
			Hint(c.Commands().ToggleLabel()).
			OnClick(func() { c.Commands().Show() }),
		ui.Spacer(),
		ui.Select("top-env", envOpts).Width(180).Selected(envSel).OnChange(func(v string) {
			_ = a.catalog.SetActiveEnv(v)
		}),
		ui.IconButton("top-settings", icons.Settings).OnClick(func() { a.openSettings(c) }),
	)
}

func (a *App) openSettings(c *ui.Ctx) {
	a.settings.Prepare()
	a.showSettingsDialog(c)
}

func (a *App) showSettingsDialog(c *ui.Ctx) {
	c.Dialogs().Show(ui.DialogOpts{
		Title:  "Settings",
		Width:  800,
		Height: 600,
		Body:   a.settings.Layout,
		OnDismiss: func() {
			a.settings.Cancel()
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { a.settings.Cancel() }},
			{Label: "Defaults", OnClick: func() {
				a.settings.LoadDefaults()
				// Dialog actions always dismiss; reopen with the defaults draft.
				a.showSettingsDialog(c)
			}},
			{Label: "Save", Primary: true, OnClick: func() {
				if err := a.settings.Save(); err != nil {
					a.showError(err)
					return
				}
				a.toast("Settings saved")
			}},
		},
	})
}

func (a *App) nav(c *ui.Ctx) ui.View {
	return ui.Nav("main-nav", ui.NavVertical, ui.NavIconTop,
		ui.NavItem{ID: "requests", Label: "Requests", Icon: icons.Send},
		ui.NavItem{ID: "environments", Label: "Envs", Icon: icons.FolderPlus},
		ui.NavItem{ID: "protofiles", Label: "Protos", Icon: icons.Code},
		ui.NavItem{ID: "workspaces", Label: "Spaces", Icon: icons.Boxes},
	).Selected(a.navIndex).OnSelectItem(func(i int, _ string) { a.navIndex = i }).Width(75)
}

func (a *App) footer(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Row(
		ui.Caption("Chapar "+version.GetAppVersion()).MarginLeft(th.Spacing.S),
		ui.Spacer(),
		ui.Button("footer-console", ui.Text("Console")).IconStart(icons.Terminal).Ghost().HoverFill().OnClick(func() {}),
		ui.Button("footer-notifications", ui.Text("Notifications")).
			IconStart(icons.Bell).
			Ghost().
			HoverFill().
			MarginRight(th.Spacing.S).
			OnClick(func() {}),
	)
}

func (a *App) Close() {
	if a.ws != nil {
		a.ws.CloseAll()
	}
	if a.executor != nil {
		_ = a.executor.Shutdown()
	}
}

func (a *App) OnKey(_ *ui.Ctx, k input.KeyEvent) bool {
	// Save/Send are registered as commands (⌘S / ⌘Enter).
	_ = k
	return false
}
