package uiv2

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/secret"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/chapar-rest/chapar/uiv2/cookieui"
	"github.com/chapar-rest/chapar/uiv2/langsrv"
	"github.com/chapar-rest/chapar/uiv2/pages"
	"github.com/chapar-rest/chapar/uiv2/scriptsrv"
	"github.com/chapar-rest/chapar/uiv2/secretui"
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
	navWorkspaces
)

type App struct {
	repo     repository.RepositoryV2
	catalog  *Catalog
	sender   *sender.Service
	settings *settings.Panel
	cookies  *cookieui.Dialog
	ws       *Workspace

	requests *pages.Requests
	envs     *pages.Environments
	spaces   *pages.Workspaces

	navIndex int
	initErr  error
	wake     func()
	uiCtx    *ui.Ctx
	scripts  *scriptsrv.Service
	lang     *langsrv.Service
	// lspPromptClosed holds languages whose install prompt the user closed.
	lspPromptClosed map[string]bool
	secrets         *secret.Manager
	console         *ConsolePanel
	notifs          NotificationHistory
	notifsOpen      bool
	sideOpen        bool
	hideNavbar      bool
}

var _ yoga.App = (*App)(nil)
var _ yoga.Closer = (*App)(nil)
var _ yoga.KeyHook = (*App)(nil)

func BuildApp() *App {
	a := &App{console: &ConsolePanel{}, sideOpen: true, lspPromptClosed: map[string]bool{}}
	a.lang = langsrv.New(nil)
	a.lang.Apply(prefs.GetGlobalConfig().Spec.LanguageServers)
	a.scripts = scriptsrv.New()
	prefs.AddGlobalConfigChangeListener(func(old, updated domain.GlobalConfig) {
		a.lang.Apply(updated.Spec.LanguageServers)
		if old.Spec.Scripting.Changed(updated.Spec.Scripting) {
			a.scripts.Restart(updated.Spec.Scripting)
		}
	})
	go func() {
		langsrv.FixPath()
		a.lang.PathReady()
	}()
	a.settings = settings.New(a.lang, a.scripts, a.installLanguageServer, func(spec domain.GlobalConfigSpec) {
		a.hideNavbar = spec.General.HideNavbar
		applyChaparAppearance(spec.General, spec.Editor)
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
	a.secrets = newSecretManager()
	repo.SetSecrets(a.secrets)
	a.settings.SetSecrets(a.secretDeps())
	a.catalog = newCatalog(repo)
	if err := a.catalog.Load(); err != nil {
		a.initErr = err
		return a
	}

	a.sender = sender.New(repo, a.catalog.RequestByID, a.catalog.CollectionByID, func(env *domain.Environment) {
		a.catalog.ReplaceEnvironment(env)
	})
	cookieStore := cookies.NewStore(repo.WorkspaceDir, nil)
	a.sender.SetCookieStore(cookieStore)
	a.cookies = cookieui.New(cookieui.Deps{
		Store:     cookieStore,
		Envs:      func() []*domain.Environment { return a.catalog.Environments },
		ActiveEnv: a.catalog.ActiveEnvironment,
		Error:     a.showError,
		Toast:     a.toast,
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
	a.spaces = pages.NewWorkspacesPage(pages.WorkspacesDeps{
		Repo:     repo,
		List:     func() []*domain.Workspace { return a.catalog.Workspaces },
		ActiveID: func() string { return a.catalog.ActiveWorkspaceID },
		Summary:  a.workspaceSummary,
		Load:     a.catalog.Load,
		Error:    a.showError,
		Use:      a.switchWorkspace,
		Renamed: func(ws *domain.Workspace) {
			// The repository finds the open workspace by its folder name,
			// which the rename just changed.
			if err := a.catalog.SetActiveWorkspace(ws); err != nil {
				a.showError(err)
			}
		},
	})

	cfg := prefs.GetGlobalConfig()
	a.hideNavbar = cfg.Spec.General.HideNavbar
	applyChaparAppearance(cfg.Spec.General, cfg.Spec.Editor)

	a.sender.SetExecutor(a.scripts)
	a.scripts.Restart(cfg.Spec.Scripting)
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

// copyToClipboard puts text on the system clipboard.
func (a *App) copyToClipboard(text string) {
	if a.uiCtx == nil {
		return
	}
	if clip := a.uiCtx.Clipboard(); clip != nil {
		clip.Set(text)
	}
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
		Lang:          a.lang,
		ManageCookies: a.openCookies,
		Secrets:       a.secrets,
		Clipboard:     a.copyToClipboard,
	}
}

func (a *App) openCookies() {
	if a.cookies != nil && a.uiCtx != nil {
		a.cookies.Show(a.uiCtx)
	}
}

func (a *App) rebuildTrees() {
	if a.requests != nil {
		a.requests.Rebuild()
	}
	if a.envs != nil {
		a.envs.Rebuild()
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
	a.notifs.Add(msg, ui.ToastInfo)
	if host := a.toasts(); host != nil {
		host.Show(msg, ui.ToastInfo, 3*time.Second)
	}
}

// reportLanguageServers logs language-server problems to the console and
// surfaces them as notifications.
func (a *App) reportLanguageServers() {
	for _, n := range a.lang.Drain() {
		variant := ui.ToastError
		if n.Warning {
			variant = ui.ToastWarning
			logger.Warn(n.Detail)
		} else {
			logger.Error(n.Detail)
		}
		if n.Missing != nil {
			a.promptInstall(*n.Missing)
			continue
		}
		a.notifs.Add(n.Message, variant)
		if host := a.toasts(); host != nil && n.Toast {
			host.Show(n.Title, variant, 6*time.Second)
		}
	}
	a.reportInstalls()
}

// reportSkippedFiles logs each data file that could not be read to the
// console, where the full path and parse error have room, and points the user
// there with one toast for the batch.
func (a *App) reportSkippedFiles() {
	if a.catalog == nil {
		return
	}
	files := a.catalog.DrainSkipped()
	if len(files) == 0 {
		return
	}
	for _, f := range files {
		logger.Warn(fmt.Sprintf("Skipped %s, it could not be read: %v", f.Path, f.Err))
	}
	title := "1 file could not be read"
	if len(files) > 1 {
		title = fmt.Sprintf("%d files could not be read", len(files))
	}
	a.notifs.Add(title+"; see the Console", ui.ToastWarning)
	if host := a.toasts(); host != nil {
		host.Notify(ui.ToastOpts{
			ID:       "skipped-files",
			Title:    title,
			Message:  "They were skipped. The Console lists each file and what is wrong with it.",
			Variant:  ui.ToastWarning,
			Actions:  []ui.ToastAction{{Label: "Show console", OnClick: a.console.Show}},
			Duration: 15 * time.Second,
		})
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

// workspaceSummary counts what the open workspace holds, for its card on the
// workspaces page.
func (a *App) workspaceSummary() string {
	plural := func(n int, one string) string {
		if n == 1 {
			return "1 " + one
		}
		return fmt.Sprintf("%d %ss", n, one)
	}
	cols := a.catalog.AllCollections()
	reqs := len(a.catalog.StandaloneRequests())
	for _, col := range cols {
		reqs += len(col.Spec.Requests)
	}
	return strings.Join([]string{
		plural(len(cols), "collection"),
		plural(reqs, "request"),
		plural(len(a.catalog.Environments), "environment"),
	}, " · ")
}

func (a *App) Body(c *ui.Ctx) ui.View {
	if a.uiCtx == nil {
		a.lang.SetWake(c.Invalidate)
		a.scripts.SetWake(c.Invalidate)
	}
	a.uiCtx = c
	a.wake = c.Invalidate
	th := c.Theme()
	a.reportLanguageServers()
	a.reportSkippedFiles()

	if a.initErr != nil {
		return ui.Column(
			ui.Title("Chapar failed to start"),
			ui.Text(a.initErr.Error()),
		).Padding(th.Spacing.L).Grow(1).Background(ui.TokenSurface)
	}

	a.registerCommands(c)

	main := ui.Column(
		a.topBar(c),
		ui.Row(
			a.nav(c),
			a.console.Wrap(c, ui.ViewOf(a.pageView(c)).Grow(1)),
		).Align(ui.AlignStretch).Grow(1),
		a.footer(c),
	).Grow(1)

	return ui.ViewOf(main).Grow(1).Background(ui.TokenSurface)
}

func (a *App) registerCommands(c *ui.Ctx) {
	cmds := []*ui.Command{
		ui.Section("Navigation"),
		ui.Cmd("nav.requests").Title("Go to Requests").Icon(icons.Send).Run(func() { a.navIndex = navRequests }),
		ui.Cmd("nav.envs").Title("Go to Environments").Icon(icons.FolderPlus).Run(func() { a.navIndex = navEnvs }),
		ui.Cmd("nav.spaces").Title("Go to Workspaces").Icon(icons.Boxes).Run(func() { a.navIndex = navWorkspaces }),
		ui.Cmd("app.settings").Title("Open Settings").Shortcut("⌘,").Icon(icons.Settings).Run(func() { a.openSettings(c) }),
		ui.Cmd("file.save").Title("Save").Shortcut("⌘S").Icon(icons.Save).Run(func() { a.ws.SaveActive() }),
		ui.Cmd("file.send").Title("Send / Invoke").Shortcut("⌘Enter").Icon(icons.Play).Run(func() { a.ws.SendActive() }),
	}
	cmds = append(cmds, a.ws.commands(c, a.workspaceVisible(), a.showWorkspace)...)
	cmds = append(cmds,
		ui.Section("Language servers"),
		ui.Cmd("lsp.restart").Title("Restart language servers").Icon(icons.RefreshCw).Run(func() {
			a.lang.Restart("")
			a.toast("Language servers restarted")
		}),
	)
	for _, s := range langsrv.Effective(prefs.GetGlobalConfig().Spec.LanguageServers) {
		l, _ := langsrv.ByID(s.Language)
		if !s.Enabled {
			continue
		}
		cmds = append(cmds, ui.Cmd("lsp.restart."+l.ID).Title("Restart "+l.Name+" language server").Icon(icons.RefreshCw).Run(func() {
			a.lang.Restart(l.ID)
			a.toast(l.Name + " language server restarted")
		}))
	}
	cmds = append(cmds, ui.Section("Open"))
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
				Title(domain.RequestDisplayName(r)).
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
			Title(domain.RequestDisplayName(r)).
			Detail("Request").
			Icon(icons.File).
			Run(func() {
				a.navIndex = navRequests
				a.ws.OpenRequest(r)
			}))
	}
	c.Commands().Register(cmds...)
}

// workspaceVisible reports whether the current page shows the tab strip.
func (a *App) workspaceVisible() bool {
	return a.navIndex == navRequests || a.navIndex == navEnvs
}

// showWorkspace switches to a page with the tab strip when none is shown.
func (a *App) showWorkspace() {
	if !a.workspaceVisible() {
		a.navIndex = navRequests
	}
}

func (a *App) pageView(c *ui.Ctx) ui.View {
	a.requests.SideOpen = a.sideOpen
	a.envs.SideOpen = a.sideOpen
	switch a.navIndex {
	case navEnvs:
		return a.envs.Layout(c)
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

	toggleIcon := icons.PanelLeft
	if a.sideOpen {
		toggleIcon = icons.PanelLeftClose
	}

	return ui.TitleBar(
		ui.IconButton("toggle-side", toggleIcon).OnClick(func() { a.toggleSide() }),
		ui.Select("top-ws", wsOpts).Width(170).Selected(wsSel).OnChange(func(v string) {
			if ws := a.catalog.WorkspaceByID(v); ws != nil {
				a.switchWorkspace(ws)
			}
		}),
		ui.Spacer(),
		ui.Button("cmd-palette", ui.Text("Commands")).Width(300).
			IconStart(icons.Search).
			Hint(c.Commands().ToggleLabel()).
			OnClick(func() { c.Commands().Show() }),
		ui.Spacer(),
		ui.IconButton("top-about", icons.Info).Tooltip("About").OnClick(func() { a.openAbout(c) }),
		ui.IconButton("top-cookies", icons.Cookie).Tooltip("Cookies").OnClick(a.openCookies),
		ui.Select("top-env", envOpts).Width(180).Selected(envSel).OnChange(func(v string) {
			_ = a.catalog.SetActiveEnv(v)
		}),
		ui.IconButton("top-settings", icons.Settings).OnClick(func() { a.openSettings(c) }),
	)
}

func (a *App) toggleSide() {
	a.sideOpen = !a.sideOpen
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
	if a.hideNavbar {
		return nil
	}

	th := c.Theme()
	return ui.Nav("main-nav", ui.NavVertical, ui.NavIconTop,
		ui.NavItem{ID: "requests", Label: "Requests", Icon: icons.Send},
		ui.NavItem{ID: "environments", Label: "Envs", Icon: icons.FolderPlus},
		ui.NavItem{ID: "workspaces", Label: "Spaces", Icon: icons.Boxes},
	).Selected(a.navIndex).OnSelectItem(func(i int, _ string) { a.navIndex = i }).
		Width(75).NavBackground(&th.ChromeMuted)
}

func (a *App) createMenuItems() []ui.MenuItem {
	return []ui.MenuItem{
		{Label: "New HTTP request", OnSelect: func() { a.navIndex = navRequests; a.requests.CreateHTTP() }},
		{Label: "New gRPC request", OnSelect: func() { a.navIndex = navRequests; a.requests.CreateGRPC() }},
		{Label: "New GraphQL request", OnSelect: func() { a.navIndex = navRequests; a.requests.CreateGraphQL() }},
		{Label: "New collection", OnSelect: func() { a.navIndex = navRequests; a.requests.CreateCollection() }},
		{Label: "New environment", OnSelect: func() { a.navIndex = navEnvs; a.envs.Create() }},
	}
}

func (a *App) footer(c *ui.Ctx) ui.View {
	th := c.Theme()
	cfg := prefs.GetGlobalConfig()
	splitIcon := icons.PanelLeft
	if cfg.Spec.General.UseHorizontalSplit {
		splitIcon = icons.PanelTop
	}
	return ui.Row(
		ui.Caption("Chapar "+version.GetAppVersion()).MarginLeft(th.Spacing.S),
		ui.Spacer(),
		ui.Button("footer-split", ui.Text("Split")).IconStart(splitIcon).Ghost().HoverFill().OnClick(func() {
			cfg := prefs.GetGlobalConfig()
			cfg.Spec.General.UseHorizontalSplit = !cfg.Spec.General.UseHorizontalSplit
			_ = prefs.UpdateGlobalConfig(cfg)
		}),
		ui.Button("footer-console", ui.Text("Console")).IconStart(icons.Terminal).Ghost().HoverFill().OnClick(func() {
			a.console.Toggle()
		}),
		ui.Popover("footer-notifications-pop",
			ui.Button("footer-notifications", ui.Text("Notifications")).
				IconStart(icons.Bell).
				Ghost().
				HoverFill().
				MarginRight(th.Spacing.S),
			a.notifs.Layout(c),
		).
			Open(a.notifsOpen).
			OnOpenChange(func(v bool) { a.notifsOpen = v }).
			Placement(ui.PlacementTop).
			Width(360).
			Height(280),
	).Background(ui.TokenChrome)
}

func (a *App) Close() {
	if a.ws != nil {
		a.ws.CloseAll()
	}
	a.scripts.Shutdown()
}

func (a *App) OnKey(_ *ui.Ctx, k input.KeyEvent) bool {
	// Save/Send are registered as commands (⌘S / ⌘Enter).
	_ = k
	return false
}

// secretDeps drives the secret key dialogs from anywhere in the app.
func (a *App) secretDeps() secretui.Deps {
	return secretui.Deps{
		Manager:   a.secrets,
		Dialogs:   a.dialogs,
		Clipboard: a.copyToClipboard,
		Toast:     a.toast,
		Error:     a.showError,
		Changed:   a.invalidate,
	}
}

// invalidate asks for a redraw after something changed outside a frame.
func (a *App) invalidate() {
	if a.wake != nil {
		a.wake()
	}
}

// newSecretManager wires the secret key store to the config directory. A
// failure here is not fatal: chapar runs, and secret values stay locked.
func newSecretManager() *secret.Manager {
	configDir, err := prefs.GetConfigDir()
	if err != nil {
		configDir = "."
	}
	return secret.New(secret.NewOSStore(), filepath.Join(configDir, secret.MetaFileName))
}
