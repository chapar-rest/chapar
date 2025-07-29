package app

import (
	"fmt"
	"time"

	"gioui.org/app"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/codegen"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/fonts"
	"github.com/chapar-rest/chapar/ui/footer"
	"github.com/chapar-rest/chapar/ui/header"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/pages/environments"
	"github.com/chapar-rest/chapar/ui/pages/protofiles"
	"github.com/chapar-rest/chapar/ui/pages/requests"
	"github.com/chapar-rest/chapar/ui/pages/settings"
	"github.com/chapar-rest/chapar/ui/pages/workspaces"
	"github.com/chapar-rest/chapar/ui/router"
	"github.com/chapar-rest/chapar/ui/widgets/fuzzysearch"
)

// App is holding the app state and configs.
type App struct {
	Window *app.Window
	Router *router.Router
	Theme  *chapartheme.Theme

	Repository repository.RepositoryV2

	Explorer *explorer.Explorer

	// TODO state should eventually be removed component should use repo directly.
	ProtoFilesState   *state.ProtoFiles
	RequestsState     *state.Requests
	EnvironmentsState *state.Environments
	WorkspacesState   *state.Workspaces

	// services
	GrpcService   *grpc.Service
	RestService   *rest.Service
	EgressService *egress.Service

	// views
	ProtoFilesView   *protofiles.View
	RequestsView     *requests.View
	EnvironmentsView *environments.View
	WorkspaceView    *workspaces.View
	SettingsView     *settings.View

	// controllers
	WorkspacesController   *workspaces.Controller
	ProtoFileController    *protofiles.Controller
	RequestsController     *requests.Controller
	SettingsController     *settings.Controller
	EnvironmentsController *environments.Controller

	// scripting executor
	Executor scripting.Executor

	// layouts
	HeaderLayout *header.Header
	FooterLayout *footer.Footer
}

func NewApp(w *app.Window, appVersion string) (*App, error) {
	fontCollection, err := fonts.Prepare()
	if err != nil {
		return nil, err
	}

	appState := prefs.GetAppState()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	th := chapartheme.New(theme, appState.Spec.DarkMode)
	explorerController := explorer.NewExplorer(w)

	// create file storage in user's home directory
	repo, err := repository.NewFilesystemV2(prefs.GetWorkspacePath(), appState.Spec.ActiveWorkspace.Name)
	if err != nil {
		return nil, err
	}

	// init state
	protoFilesState := state.NewProtoFiles(repo)
	requestsState := state.NewRequests(repo)
	environmentsState := state.NewEnvironments(repo)
	workspacesState, err := state.NewWorkspaces(repo)
	if err != nil {
		return nil, err
	}

	// init services
	grpcService := grpc.NewService(appVersion, requestsState, environmentsState, protoFilesState)
	restService := rest.New(requestsState, environmentsState, appVersion)
	egressService := egress.New(requestsState, environmentsState, restService, grpcService, nil)

	// init views
	workspaceView := workspaces.NewView()
	protoFilesView := protofiles.NewView()
	settingsView := settings.NewView(w, th)
	environmentsView := environments.NewView(w, th)
	requestsView := requests.NewView(w, th, explorerController)

	// init controllers
	workspacesController := workspaces.NewController(workspaceView, workspacesState, repo)
	if err := workspacesController.LoadData(); err != nil {
		return nil, err
	}
	protoFileController := protofiles.NewController(protoFilesView, protoFilesState, repo, explorerController)
	if err := protoFileController.LoadData(); err != nil {
		return nil, err
	}
	requestsController := requests.NewController(requestsView, repo, requestsState, environmentsState, explorerController, egressService, grpcService)
	if err := requestsController.LoadData(); err != nil {
		return nil, err
	}
	// settings view loads its own data from prefs packages.
	settingsController := settings.NewController(settingsView)

	// init environments controller
	environmentsController := environments.NewController(environmentsView, repo, environmentsState, explorerController)

	// layouts
	headerLayout := header.NewHeader(w, environmentsState, workspacesState, th)
	footerLayout := &footer.Footer{AppVersion: appVersion}

	r := router.New(headerLayout, footerLayout, th)
	r.Register(router.RequestsTag, requestsView)
	r.Register(router.EnvironmentsTag, environmentsView)
	r.Register(router.ProtoFilesTag, protoFilesView)
	r.Register(router.WorkspacesTag, workspaceView)
	r.Register(router.SettingsTag, settingsView)

	// init notification system
	// TODO should be part of the app state.
	notifications.New(w)

	chApp := &App{
		Window:                 w,
		Router:                 r,
		Theme:                  th,
		Repository:             repo,
		Explorer:               explorerController,
		ProtoFilesState:        protoFilesState,
		RequestsState:          requestsState,
		EnvironmentsState:      environmentsState,
		WorkspacesState:        workspacesState,
		GrpcService:            grpcService,
		RestService:            restService,
		EgressService:          egressService,
		ProtoFilesView:         protoFilesView,
		RequestsView:           requestsView,
		EnvironmentsView:       environmentsView,
		WorkspaceView:          workspaceView,
		SettingsView:           settingsView,
		WorkspacesController:   workspacesController,
		ProtoFileController:    protoFileController,
		RequestsController:     requestsController,
		SettingsController:     settingsController,
		EnvironmentsController: environmentsController,
		Executor:               nil, // will be initialized in initiateScripting
		HeaderLayout:           headerLayout,
		FooterLayout:           footerLayout,
	}

	return chApp, nil
}

func Init(chApp *App) error {
	// init executor in a separate goroutine
	initExecutor := func() {
		if exec := initiateScripting(); exec != nil {
			chApp.Executor = exec
			chApp.EgressService.SetExecutor(exec)
		}
	}

	go initExecutor()

	// listen for changes in scripting config
	prefs.AddGlobalConfigChangeListener(func(old, updated domain.GlobalConfig) {
		if old.Spec.Scripting.Changed(updated.Spec.Scripting) {
			if updated.Spec.Scripting.Enabled {
				go initExecutor()
			} else if old.Spec.Scripting.Enabled && !updated.Spec.Scripting.Enabled {
				go stopScripting(chApp.Executor)
			}
		}
	})

	// listen for changes in the active environment
	chApp.EnvironmentsState.AddActiveEnvironmentChangeListener(codegen.DefaultService.OnActiveEnvironmentChange)

	// header setup
	chApp.HeaderLayout.LoadWorkspaces(chApp.WorkspacesState.GetWorkspaces())
	chApp.HeaderLayout.SetSearchDataLoader(chApp.searchDataLoader)
	chApp.HeaderLayout.SetOnSearchResultSelect(chApp.onSelectSearchResult)
	chApp.HeaderLayout.OnSelectedEnvChanged = chApp.onSelectedEnvChanged
	chApp.HeaderLayout.OnSelectedWorkspaceChanged = chApp.onWorkspaceChanged

	chApp.EnvironmentsState.AddEnvironmentChangeListener(func(environment *domain.Environment, source state.Source, action state.Action) {
		chApp.HeaderLayout.LoadEnvs(chApp.EnvironmentsState.GetEnvironments())
	})

	chApp.WorkspacesState.AddWorkspaceChangeListener(func(workspace *domain.Workspace, source state.Source, action state.Action) {
		chApp.HeaderLayout.LoadWorkspaces(chApp.WorkspacesState.GetWorkspaces())
	})

	chApp.HeaderLayout.OnThemeSwitched = chApp.onThemeChange

	return chApp.loadWorkspace()
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
		a.Router.SwitchTo(router.EnvironmentsTag)
	case domain.KindRequest:
		a.RequestsController.OpenRequest(result.Item.Identifier)
		a.Router.SwitchTo(router.RequestsTag)
	case domain.KindCollection:
		a.RequestsController.OpenCollection(result.Item.Identifier)
		a.Router.SwitchTo(router.RequestsTag)
	case domain.KindProtoFile:
		a.Router.SwitchTo(router.ProtoFilesTag)
	case domain.KindWorkspace:
		a.Router.SwitchTo(router.WorkspacesTag)
	}
}

func (a *App) showError(err error) {
	a.Router.SetMessageDialog(modals.Message{
		Title: "Error",
		Body:  err.Error(),
		Type:  modals.MessageTypeErr,
		OKBtn: widget.Clickable{},
	}, a.Theme)
}

func (a *App) onSelectedEnvChanged(env *domain.Environment) error {
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
		return fmt.Errorf("failed to update app state, %w", err)
	}

	if env != nil {
		a.EnvironmentsState.SetActiveEnvironment(env)
	} else {
		a.EnvironmentsState.ClearActiveEnvironment()
	}

	return nil
}

func (a *App) onWorkspaceChanged(ws *domain.Workspace) error {
	appState := prefs.GetAppState()
	appState.Spec.ActiveWorkspace = &domain.ActiveWorkspace{
		ID:   ws.MetaData.ID,
		Name: ws.MetaData.Name,
	}

	if err := prefs.UpdateAppState(appState); err != nil {
		return fmt.Errorf("failed to update app state, %w", err)
	}

	a.Repository.SetActiveWorkspace(ws.GetName())
	a.WorkspacesState.SetActiveWorkspace(ws)

	if err := a.loadWorkspace(); err != nil {
		return fmt.Errorf("failed to load data, %w", err)
	}
	return nil
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

func (a *App) onThemeChange(isDark bool) error {
	a.Theme.Switch(isDark)

	appState := prefs.GetAppState()
	appState.Spec.DarkMode = isDark
	if err := prefs.UpdateAppState(appState); err != nil {
		return fmt.Errorf("failed to update app state, %w", err)
	}
	return nil
}
