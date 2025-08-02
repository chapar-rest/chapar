package app

import (
	"fmt"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/internal/codegen"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/navigator"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/widgets/fuzzysearch"
)

// App is holding the app state and configs.
type App struct {
	*app.Window
	*Base
	*BaseLayout
}

func NewApp(w *app.Window, appVersion string) (*App, error) {
	// init notification system
	notifications.New(w)

	navi := navigator.New()
	base, err := NewBase(appVersion, w, navi)
	if err != nil {
		return nil, err
	}

	baseLayout, err := NewBaseLayout(base)
	if err != nil {
		return nil, err
	}
	navi.Register(baseLayout.RequestsView)
	navi.Register(baseLayout.EnvironmentsView)
	navi.Register(baseLayout.ProtoFilesView)
	navi.Register(baseLayout.WorkspaceView)
	navi.Register(baseLayout.SettingsView)

	out := &App{
		Window:     w,
		Base:       base,
		BaseLayout: baseLayout,
	}

	// init executor in a separate goroutine
	initExecutor := func() {
		if exec := initiateScripting(); exec != nil {
			base.Executor = exec
			base.EgressService.SetExecutor(exec)
		}
	}

	go initExecutor()

	// listen for changes in scripting config
	prefs.AddGlobalConfigChangeListener(func(old, updated domain.GlobalConfig) {
		if old.Spec.Scripting.Changed(updated.Spec.Scripting) {
			if updated.Spec.Scripting.Enabled {
				go initExecutor()
			} else if old.Spec.Scripting.Enabled && !updated.Spec.Scripting.Enabled {
				go stopScripting(base.Executor)
			}
		}
	})

	// listen for changes in the active environment
	base.EnvironmentsState.AddActiveEnvironmentChangeListener(codegen.DefaultService.OnActiveEnvironmentChange)

	// header setup
	baseLayout.HeaderLayout.LoadWorkspaces(base.WorkspacesState.GetWorkspaces())
	baseLayout.HeaderLayout.SetSearchDataLoader(out.searchDataLoader)
	baseLayout.HeaderLayout.SetOnSearchResultSelect(out.onSelectSearchResult)
	baseLayout.HeaderLayout.OnSelectedEnvChanged = out.onSelectedEnvChanged
	baseLayout.HeaderLayout.OnSelectedWorkspaceChanged = out.onWorkspaceChanged

	base.EnvironmentsState.AddEnvironmentChangeListener(func(environment *domain.Environment, source state.Source, action state.Action) {
		baseLayout.HeaderLayout.LoadEnvs(base.EnvironmentsState.GetEnvironments())
	})

	base.WorkspacesState.AddWorkspaceChangeListener(func(workspace *domain.Workspace, source state.Source, action state.Action) {
		baseLayout.HeaderLayout.LoadWorkspaces(base.WorkspacesState.GetWorkspaces())
	})

	baseLayout.HeaderLayout.OnThemeSwitched = out.onThemeChange

	return out, out.loadWorkspace()
}

func initiateScripting() scripting.Executor {
	config := prefs.GetGlobalConfig().Spec.Scripting

	executor, err := scripting.GetExecutor(config.Language, config)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get scripting executor: %v", err))
		return nil
	}

	notifications.Send(fmt.Sprintf("Initializing %s script executor...", executor.Name()), notifications.NotificationTypeInfo, 5*time.Second)
	logger.Info(fmt.Sprintf("Initializing %s executor", executor.Name()))

	if err := executor.Init(config); err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize %s executor: %vAfter fixing the issue, enable and disable the scripting from Settings->Scripting to trigger initiation again.", executor.Name(), err))
		notifications.Send(fmt.Sprintf("Failed to initialize %s executor, check console for errors", executor.Name()), notifications.NotificationTypeError, 10*time.Second)
		return nil
	}

	notifications.Send(fmt.Sprintf("%s executor initialized successfully", executor.Name()), notifications.NotificationTypeInfo, 5*time.Second)
	return executor
}

func stopScripting(executor scripting.Executor) {
	if executor == nil {
		return
	}

	logger.Info(fmt.Sprintf("Stopping %s executor", executor.Name()))
	if err := executor.Shutdown(); err != nil {
		logger.Error(fmt.Sprintf("Failed to stop %s executor: %s", executor.Name(), err))
	}
}

func (a *App) searchDataLoader() []fuzzysearch.Item {
	envs, err := a.Repository.LoadEnvironments()
	if err != nil {
		a.showError(fmt.Errorf("failed to load environments, %w", err))
		return nil
	}

	cols, err := a.Repository.LoadCollections()
	if err != nil {
		a.showError(fmt.Errorf("failed to load collections, %w", err))
		return nil
	}

	protoFiles, err := a.Repository.LoadProtoFiles()
	if err != nil {
		a.showError(fmt.Errorf("failed to load proto files, %w", err))
		return nil
	}

	reqs, err := a.Repository.LoadRequests()
	if err != nil {
		a.showError(fmt.Errorf("failed to load requests, %w", err))
		return nil
	}

	items := make([]fuzzysearch.Item, 0)
	for _, env := range envs {
		items = append(items, fuzzysearch.Item{
			Identifier: env.MetaData.ID,
			Kind:       domain.KindEnv,
			Title:      env.MetaData.Name,
		})
	}

	for _, col := range cols {
		items = append(items, fuzzysearch.Item{
			Identifier: col.MetaData.ID,
			Kind:       domain.KindCollection,
			Title:      col.MetaData.Name,
		})

		for _, req := range col.Spec.Requests {
			items = append(items, fuzzysearch.Item{
				Identifier: req.MetaData.ID,
				Kind:       domain.KindRequest,
				Title:      req.MetaData.Name,
			})
		}
	}

	for _, protoFile := range protoFiles {
		items = append(items, fuzzysearch.Item{
			Identifier: protoFile.MetaData.ID,
			Kind:       domain.KindProtoFile,
			Title:      protoFile.MetaData.Name,
		})
	}

	for _, req := range reqs {
		items = append(items, fuzzysearch.Item{
			Identifier: req.MetaData.ID,
			Kind:       domain.KindRequest,
			Title:      req.MetaData.Name,
		})
	}

	return items
}

func (a *App) onSelectSearchResult(result *fuzzysearch.SearchResult) {
	switch result.Item.Kind {
	case domain.KindEnv:
		a.EnvironmentsController.OpenEnvironment(result.Item.Identifier)
		a.Navigator.SwitchTo(navigator.EnvironmentsPageId)
	case domain.KindRequest:
		a.RequestsController.OpenRequest(result.Item.Identifier)
		a.Navigator.SwitchTo(navigator.RequestsPageId)
	case domain.KindCollection:
		a.RequestsController.OpenCollection(result.Item.Identifier)
		a.Navigator.SwitchTo(navigator.RequestsPageId)
	case domain.KindProtoFile:
		a.Navigator.SwitchTo(navigator.ProtoFilesPageId)
	case domain.KindWorkspace:
		a.Navigator.SwitchTo(navigator.WorkspacesPageId)
	}
}

func (a *App) showError(err error) {
	a.Base.SetModal(func(gtx layout.Context) layout.Dimensions {
		m := &modals.Message{
			Title: "Error",
			Body:  err.Error(),
			Type:  modals.MessageTypeErr,
			OKBtn: widget.Clickable{},
		}
		return m.Layout(gtx, a.Theme)
	})
}

func (a *App) onSelectedEnvChanged(env *domain.Environment) {
	appState := prefs.GetAppState()
	if env != nil {
		if appState.Spec.SelectedEnvironment == nil {
			appState.Spec.SelectedEnvironment = &domain.SelectedEnvironment{}
		}

		appState.Spec.SelectedEnvironment.ID = env.MetaData.ID
		appState.Spec.SelectedEnvironment.Name = env.MetaData.Name
	} else {
		appState.Spec.SelectedEnvironment = nil
	}

	if err := prefs.UpdateAppState(appState); err != nil {
		a.showError(err)
		return
	}

	if env != nil {
		a.EnvironmentsState.SetActiveEnvironment(env)
	} else {
		a.EnvironmentsState.ClearActiveEnvironment()
	}
}

func (a *App) onWorkspaceChanged(ws *domain.Workspace) {
	appState := prefs.GetAppState()
	appState.Spec.ActiveWorkspace = &domain.ActiveWorkspace{
		ID:   ws.MetaData.ID,
		Name: ws.MetaData.Name,
	}

	if err := prefs.UpdateAppState(appState); err != nil {
		a.showError(err)
		return
	}

	a.Repository.SetActiveWorkspace(ws.GetName())
	a.WorkspacesState.SetActiveWorkspace(ws)

	if err := a.loadWorkspace(); err != nil {
		a.showError(err)
	}
}

func (a *App) loadWorkspace() error {
	appState := prefs.GetAppState()

	a.HeaderLayout.SetTheme(appState.Spec.DarkMode)
	a.Theme.Switch(appState.Spec.DarkMode)

	if err := a.EnvironmentsController.LoadData(); err != nil {
		return err
	}

	a.HeaderLayout.LoadEnvs(a.EnvironmentsState.GetEnvironments())

	if appState.Spec.SelectedEnvironment != nil {
		if selectedEnv := a.EnvironmentsState.GetEnvironment(appState.Spec.SelectedEnvironment.ID); selectedEnv != nil {
			a.EnvironmentsState.SetActiveEnvironment(selectedEnv)
			a.HeaderLayout.SetSelectedEnvironment(a.EnvironmentsState.GetActiveEnvironment())
		}
	}

	if appState.Spec.ActiveWorkspace != nil {
		if selectedWs := a.WorkspacesState.GetWorkspace(appState.Spec.ActiveWorkspace.ID); selectedWs != nil {
			a.WorkspacesState.SetActiveWorkspace(selectedWs)
			a.HeaderLayout.SetSelectedWorkspace(a.WorkspacesState.GetActiveWorkspace())
		}
	}

	return a.RequestsController.LoadData()
}

func (a *App) onThemeChange(isDark bool) {
	a.Theme.Switch(isDark)

	appState := prefs.GetAppState()
	appState.Spec.DarkMode = isDark
	if err := prefs.UpdateAppState(appState); err != nil {
		a.showError(err)
	}
}
