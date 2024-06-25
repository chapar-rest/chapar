package app

import (
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/modal"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/notify"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/fonts"
	"github.com/chapar-rest/chapar/ui/pages/console"
	"github.com/chapar-rest/chapar/ui/pages/environments"
	"github.com/chapar-rest/chapar/ui/pages/requests"
	"github.com/chapar-rest/chapar/ui/pages/workspaces"
)

type UI struct {
	Theme  *chapartheme.Theme
	window *app.Window

	sideBar *Sidebar
	header  *Header

	consolePage *console.Console

	environmentsView *environments.View
	requestsView     *requests.View
	workspacesView   *workspaces.View

	environmentsController *environments.Controller
	requestsController     *requests.Controller
	workspacesController   *workspaces.Controller

	environmentsState *state.Environments
	requestsState     *state.Requests
	workspacesState   *state.Workspaces

	repo repository.Repository
}

// New creates a new UI using the Go Fonts.
func New(w *app.Window) (*UI, error) {
	u := &UI{
		window: w,
	}

	fontCollection, err := fonts.Prepare()
	if err != nil {
		return nil, err
	}

	repo, err := repository.NewFilesystem()
	if err != nil {
		return nil, err
	}

	u.repo = repo

	u.workspacesView = workspaces.NewView()
	u.workspacesState = state.NewWorkspaces(repo)
	u.workspacesController = workspaces.NewController(u.workspacesView, u.workspacesState, repo)
	if err := u.workspacesController.LoadData(); err != nil {
		return nil, err
	}

	u.environmentsState = state.NewEnvironments(repo)
	u.requestsState = state.NewRequests(repo)

	grpcService := grpc.NewService(u.requestsState, u.environmentsState)

	restService := rest.New(u.requestsState, u.environmentsState)
	explorerController := explorer.NewExplorer(w)

	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	// lest assume is dark theme, we will switch it later
	u.Theme = chapartheme.New(theme, true)
	// console need to be initialized before other pages as its listening for logs
	u.consolePage = console.New()

	u.header = NewHeader(u.environmentsState, u.workspacesState, u.Theme)
	u.sideBar = NewSidebar(u.Theme)

	u.header.LoadWorkspaces(u.workspacesState.GetWorkspaces())

	//
	u.environmentsView = environments.NewView(u.Theme)
	u.environmentsController = environments.NewController(u.environmentsView, repo, u.environmentsState, explorerController)
	u.environmentsState.AddEnvironmentChangeListener(func(environment *domain.Environment, source state.Source, action state.Action) {
		u.header.LoadEnvs(u.environmentsState.GetEnvironments())
	})

	u.header.OnSelectedEnvChanged = func(env *domain.Environment) {
		preferences, err := u.repo.ReadPreferencesData()
		if err != nil {
			fmt.Println("failed to read preferences: ", err)
			return
		}

		preferences.Spec.SelectedEnvironment.ID = env.MetaData.ID
		preferences.Spec.SelectedEnvironment.Name = env.MetaData.Name
		if err := repo.UpdatePreferences(preferences); err != nil {
			fmt.Println("failed to update preferences: ", err)
		}

		u.environmentsState.SetActiveEnvironment(env)
	}

	u.requestsView = requests.NewView(w, u.Theme)
	u.requestsController = requests.NewController(u.requestsView, repo, u.requestsState, u.environmentsState, explorerController, restService, grpcService)

	u.header.OnSelectedWorkspaceChanged = func(ws *domain.Workspace) {
		if err := repo.SetActiveWorkspace(ws); err != nil {
			fmt.Println("failed to set active workspace: ", err)
			return
		}
		u.workspacesState.SetActiveWorkspace(ws)

		if err := u.load(); err != nil {
			fmt.Println("failed to load data: ", err)
		}
	}

	u.workspacesState.AddWorkspaceChangeListener(func(workspace *domain.Workspace, source state.Source, action state.Action) {
		u.header.LoadWorkspaces(u.workspacesState.GetWorkspaces())
	})

	u.header.OnThemeSwitched = func(isDark bool) {
		u.Theme.Switch(isDark)

		preferences, err := u.repo.ReadPreferencesData()
		if err != nil {
			fmt.Println("failed to read preferences: ", err)
			return
		}

		preferences.Spec.DarkMode = isDark
		if err := repo.UpdatePreferences(preferences); err != nil {
			fmt.Println("failed to update preferences: ", err)
		}
	}

	// u.notification = &widgets.Notification{}
	return u, u.load()
}

func (u *UI) load() error {
	preferences, err := u.repo.ReadPreferencesData()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			preferences = domain.NewPreferences()
			if err := u.repo.UpdatePreferences(preferences); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	config, err := u.repo.GetConfig()
	if err != nil {
		return err
	}

	u.header.SetTheme(preferences.Spec.DarkMode)

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
	layout.Stack{Alignment: layout.S}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical, Spacing: 0}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// return layout.Dimensions{}
					return u.header.Layout(gtx, u.Theme)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: 0}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.sideBar.Layout(gtx, u.Theme)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							switch u.sideBar.SelectedIndex() {
							case 0:
								return u.requestsView.Layout(gtx, u.Theme)
							case 1:
								return u.environmentsView.Layout(gtx, u.Theme)
							case 2:
								return u.workspacesView.Layout(gtx, u.Theme)
								// case 4:
								//	return u.consolePage.Layout(gtx, u.Theme)
							}
							return layout.Dimensions{}
						}),
					)
				}),
			)
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if modal.Visible() {
				macro := op.Record(gtx.Ops)
				dims := modal.Layout(gtx, u.Theme.Theme)
				op.Defer(gtx.Ops, macro.Stop())
				return dims
			}

			return notify.NotificationController.Layout(gtx, u.Theme)
		}),
	)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
