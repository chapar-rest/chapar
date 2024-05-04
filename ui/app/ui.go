package app

import (
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/chapar-rest/chapar/ui/pages/workspaces"

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
	"github.com/chapar-rest/chapar/ui/widgets"
)

type UI struct {
	Theme  *chapartheme.Theme
	window *app.Window

	sideBar *Sidebar
	header  *Header

	consolePage  *console.Console
	notification *widgets.Notification

	environmentsView *environments.View
	requestsView     *requests.View
	workspacesView   *workspaces.View

	tipsOpen bool
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

	environmentsState := state.NewEnvironments(repo)
	requestsState := state.NewRequests(repo)

	preferences, err := repo.ReadPreferencesData()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			preferences = domain.NewPreferences()
			if err := repo.UpdatePreferences(preferences); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	restService := rest.New(requestsState, environmentsState)
	explorerController := explorer.NewExplorer(w)

	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	u.Theme = chapartheme.New(theme, preferences.Spec.DarkMode)
	// console need to be initialized before other pages as its listening for logs
	u.consolePage = console.New()

	u.header = NewHeader(environmentsState, u.Theme)
	u.sideBar = NewSidebar(u.Theme)
	//
	u.environmentsView = environments.NewView(u.Theme)
	envController := environments.NewController(u.environmentsView, repo, environmentsState, explorerController)
	if err := envController.LoadData(); err != nil {
		return nil, err
	}
	//
	u.header.LoadEnvs(environmentsState.GetEnvironments())
	environmentsState.AddEnvironmentChangeListener(func(environment *domain.Environment, source state.Source, action state.Action) {
		u.header.LoadEnvs(environmentsState.GetEnvironments())
	})
	//
	if selectedEnv := environmentsState.GetEnvironment(preferences.Spec.SelectedEnvironment.ID); selectedEnv != nil {
		environmentsState.SetActiveEnvironment(selectedEnv)
		u.header.SetSelectedEnvironment(environmentsState.GetActiveEnvironment())
	}
	//
	u.header.OnSelectedEnvChanged = func(env *domain.Environment) {
		preferences.Spec.SelectedEnvironment.ID = env.MetaData.ID
		preferences.Spec.SelectedEnvironment.Name = env.MetaData.Name
		if err := repo.UpdatePreferences(preferences); err != nil {
			fmt.Println("failed to update preferences: ", err)
		}

		environmentsState.SetActiveEnvironment(env)
	}
	//
	u.header.SetTheme(preferences.Spec.DarkMode)
	u.header.OnThemeSwitched = func(isDark bool) {
		u.Theme.Switch(isDark)

		preferences.Spec.DarkMode = isDark
		if err := repo.UpdatePreferences(preferences); err != nil {
			fmt.Println("failed to update preferences: ", err)
		}
	}

	u.requestsView = requests.NewView(w, u.Theme)
	reqController := requests.NewController(u.requestsView, repo, requestsState, environmentsState, explorerController, restService)
	if err := reqController.LoadData(); err != nil {
		return nil, err
	}

	u.workspacesView = workspaces.NewView()
	workspaceController := workspaces.NewController(u.workspacesView, state.NewWorkspaces(repo), repo)
	if err := workspaceController.LoadData(); err != nil {
		return nil, err
	}

	u.notification = &widgets.Notification{}
	return u, nil
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
			u.Layout(gtx, gtx.Constraints.Max.X)
			// render and handle the operations from the UI.
			e.Frame(gtx.Ops)
		// this is sent when the application is closed.
		case app.DestroyEvent:
			return e.Err
		}
	}
}

// Layout displays the main program layout.
func (u *UI) Layout(gtx layout.Context, windowWidth int) layout.Dimensions {
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
			return notify.NotificationController.Layout(gtx, u.Theme, windowWidth)
		}),
	)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
