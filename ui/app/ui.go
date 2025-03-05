package app

import (
	"fmt"
	"image"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/codegen"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/fonts"
	"github.com/chapar-rest/chapar/ui/pages/console"
	"github.com/chapar-rest/chapar/ui/pages/environments"
	"github.com/chapar-rest/chapar/ui/pages/protofiles"
	"github.com/chapar-rest/chapar/ui/pages/requests"
	"github.com/chapar-rest/chapar/ui/pages/workspaces"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/fuzzysearch"
)

type UI struct {
	Theme  *chapartheme.Theme
	window *app.Window

	sideBar *Sidebar
	header  *Header

	modal *widgets.MessageModal

	currentPage int

	consolePage *console.Console

	environmentsView *environments.View
	requestsView     *requests.View
	workspacesView   *workspaces.View
	protoFilesView   *protofiles.View

	environmentsController *environments.Controller
	requestsController     *requests.Controller
	workspacesController   *workspaces.Controller
	protoFilesController   *protofiles.Controller

	environmentsState *state.Environments
	requestsState     *state.Requests
	workspacesState   *state.Workspaces
	protoFilesState   *state.ProtoFiles

	repo repository.Repository
}

// New creates a new UI using the Go Fonts.
func New(w *app.Window, appVersion string) (*UI, error) {
	u := &UI{
		window: w,
	}

	fontCollection, err := fonts.Prepare()
	if err != nil {
		return nil, err
	}

	// create file storage in user's home directory
	repo, err := repository.NewFilesystem(repository.DefaultConfigDir, "" /* baseDir */)
	if err != nil {
		return nil, err
	}

	explorerController := explorer.NewExplorer(w)

	u.repo = repo

	preferences, err := u.repo.ReadPreferences()
	if err != nil {
		return nil, fmt.Errorf("failed to read preferences, %w", err)
	}

	u.workspacesView = workspaces.NewView()
	u.workspacesState = state.NewWorkspaces(repo)
	u.workspacesController = workspaces.NewController(u.workspacesView, u.workspacesState, repo)
	if err := u.workspacesController.LoadData(); err != nil {
		return nil, err
	}

	u.environmentsState = state.NewEnvironments(repo)
	u.requestsState = state.NewRequests(repo)

	// listen for changes in the active environment
	u.environmentsState.AddActiveEnvironmentChangeListener(codegen.DefaultService.OnActiveEnvironmentChange)

	//
	u.protoFilesView = protofiles.NewView()
	u.protoFilesState = state.NewProtoFiles(repo)
	u.protoFilesController = protofiles.NewController(u.protoFilesView, u.protoFilesState, repo, explorerController)
	if err := u.protoFilesController.LoadData(); err != nil {
		return nil, err
	}

	grpcService := grpc.NewService(appVersion, u.requestsState, u.environmentsState, u.protoFilesState)
	restService := rest.New(u.requestsState, u.environmentsState)

	egressService := egress.New(u.requestsState, u.environmentsState, restService, grpcService)

	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	u.Theme = chapartheme.New(theme, preferences.Spec.DarkMode)
	// console need to be initialized before other pages as its listening for logs
	u.consolePage = console.New()

	u.header = NewHeader(w, u.environmentsState, u.workspacesState, u.Theme)
	u.header.SetSearchDataLoader(u.searchDataLoader)
	u.header.SetOnSearchResultSelect(u.onSelectSearchResult)

	u.sideBar = NewSidebar(u.Theme, appVersion)

	u.sideBar.OnSelectedChanged = func(index int) {
		u.currentPage = index
	}

	u.header.LoadWorkspaces(u.workspacesState.GetWorkspaces())

	//
	u.environmentsView = environments.NewView(w, u.Theme)
	u.environmentsController = environments.NewController(u.environmentsView, repo, u.environmentsState, explorerController)
	u.environmentsState.AddEnvironmentChangeListener(func(environment *domain.Environment, source state.Source, action state.Action) {
		u.header.LoadEnvs(u.environmentsState.GetEnvironments())
	})

	u.header.OnSelectedEnvChanged = u.onSelectedEnvChanged

	u.requestsView = requests.NewView(w, u.Theme, explorerController)
	u.requestsController = requests.NewController(u.requestsView, repo, u.requestsState, u.environmentsState, explorerController, egressService, grpcService)

	u.header.OnSelectedWorkspaceChanged = u.onWorkspaceChanged

	u.workspacesState.AddWorkspaceChangeListener(func(workspace *domain.Workspace, source state.Source, action state.Action) {
		u.header.LoadWorkspaces(u.workspacesState.GetWorkspaces())
	})

	u.header.OnThemeSwitched = u.onThemeChange

	return u, u.load()
}

func (u *UI) showError(err error) {
	u.modal = widgets.NewMessageModal("Error", err.Error(), widgets.MessageModalTypeErr, func(_ string) {
		u.modal.Hide()
	}, widgets.ModalOption{Text: "Ok"})
	u.modal.Show()
}

func (u *UI) searchDataLoader() []fuzzysearch.Item {
	envs, err := u.repo.LoadEnvironments()
	if err != nil {
		u.showError(fmt.Errorf("failed to load environments, %w", err))
		return nil
	}

	cols, err := u.repo.LoadCollections()
	if err != nil {
		u.showError(fmt.Errorf("failed to load collections, %w", err))
		return nil
	}

	protoFiles, err := u.repo.LoadProtoFiles()
	if err != nil {
		u.showError(fmt.Errorf("failed to load proto files, %w", err))
		return nil
	}

	reqs, err := u.repo.LoadRequests()
	if err != nil {
		u.showError(fmt.Errorf("failed to load requests, %w", err))
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

func (u *UI) onSelectSearchResult(result *fuzzysearch.SearchResult) {
	switch result.Item.Kind {
	case domain.KindEnv:
		u.environmentsController.OpenEnvironment(result.Item.Identifier)
		u.currentPage = 1
	case domain.KindRequest:
		u.requestsController.OpenRequest(result.Item.Identifier)
		u.currentPage = 0
	case domain.KindCollection:
		u.requestsController.OpenCollection(result.Item.Identifier)
		u.currentPage = 0
	case domain.KindProtoFile:
		u.currentPage = 3
	case domain.KindWorkspace:
		u.currentPage = 2
	}
}

func (u *UI) onWorkspaceChanged(ws *domain.Workspace) error {
	if err := u.repo.SetActiveWorkspace(ws); err != nil {
		return fmt.Errorf("failed to set active workspace, %w", err)
	}
	u.workspacesState.SetActiveWorkspace(ws)

	if err := u.load(); err != nil {
		return fmt.Errorf("failed to load data, %w", err)
	}
	return nil
}

func (u *UI) onSelectedEnvChanged(env *domain.Environment) error {
	preferences, err := u.repo.ReadPreferences()
	if err != nil {
		return fmt.Errorf("failed to read preferences, %w", err)
	}

	if env != nil {
		preferences.Spec.SelectedEnvironment.ID = env.MetaData.ID
		preferences.Spec.SelectedEnvironment.Name = env.MetaData.Name
	} else {
		preferences.Spec.SelectedEnvironment.ID = ""
		preferences.Spec.SelectedEnvironment.Name = ""
	}

	if err := u.repo.UpdatePreferences(preferences); err != nil {
		return fmt.Errorf("failed to update preferences, %w", err)
	}

	if env != nil {
		u.environmentsState.SetActiveEnvironment(env)
	} else {
		u.environmentsState.ClearActiveEnvironment()
	}

	return nil
}

func (u *UI) onThemeChange(isDark bool) error {
	u.Theme.Switch(isDark)

	preferences, err := u.repo.ReadPreferences()
	if err != nil {
		return fmt.Errorf("failed to read preferences, %w", err)
	}

	preferences.Spec.DarkMode = isDark
	if err := u.repo.UpdatePreferences(preferences); err != nil {
		return fmt.Errorf("failed to update preferences, %w", err)
	}
	return nil
}

func (u *UI) load() error {
	preferences, err := u.repo.ReadPreferences()
	if err != nil {
		return err
	}

	config, err := u.repo.GetConfig()
	if err != nil {
		return err
	}

	u.header.SetTheme(preferences.Spec.DarkMode)
	u.Theme.Switch(preferences.Spec.DarkMode)

	if err := u.environmentsController.LoadData(); err != nil {
		return err
	}

	u.header.LoadEnvs(u.environmentsState.GetEnvironments())

	if selectedEnv := u.environmentsState.GetEnvironment(preferences.Spec.SelectedEnvironment.ID); selectedEnv != nil {
		u.environmentsState.SetActiveEnvironment(selectedEnv)
		u.header.SetSelectedEnvironment(u.environmentsState.GetActiveEnvironment())
	}

	if selectedWs := u.workspacesState.GetWorkspace(config.Spec.ActiveWorkspace.ID); selectedWs != nil {
		u.workspacesState.SetActiveWorkspace(selectedWs)
		u.header.SetSelectedWorkspace(u.workspacesState.GetActiveWorkspace())
	}

	return u.requestsController.LoadData()
}

func (u *UI) Run() error {
	// ops are the operations from the UI
	var ops op.Ops

	for {
		switch e := u.window.Event().(type) {
		// this is sent when the application should re-render.
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			// render and handle UI.
			u.Layout(gtx)
			// render and handle the operations from the UI.
			e.Frame(gtx.Ops)
		// this is sent when the application is closed.
		case app.DestroyEvent:
			return e.Err
		}
	}
}

// Layout displays the main program layout.
func (u *UI) Layout(gtx layout.Context) layout.Dimensions {
	// set the background color
	macro := op.Record(gtx.Ops)
	rect := image.Rectangle{
		Max: image.Point{
			X: gtx.Constraints.Max.X,
			Y: gtx.Constraints.Max.Y,
		},
	}
	paint.FillShape(gtx.Ops, u.Theme.Palette.Bg, clip.Rect(rect).Op())
	background := macro.Stop()

	background.Add(gtx.Ops)

	u.modal.Layout(gtx, u.Theme)

	return layout.Flex{Axis: layout.Vertical, Spacing: 0}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.header.Layout(gtx, u.Theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: 0}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.sideBar.Layout(gtx, u.Theme)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					switch u.currentPage {
					case 0:
						return u.requestsView.Layout(gtx, u.Theme)
					case 1:
						return u.environmentsView.Layout(gtx, u.Theme)
					case 2:
						return u.workspacesView.Layout(gtx, u.Theme)
					case 3:
						return u.protoFilesView.Layout(gtx, u.Theme)
						// case 4:
						//	return u.consolePage.Layout(gtx, u.Theme)
					}
					return layout.Dimensions{}
				}),
			)
		}),
	)
}
