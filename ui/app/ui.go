package app

import (
	"errors"
	"fmt"
	"image/color"
	"os"

	"github.com/mirzakhany/chapar/internal/rest"

	"github.com/mirzakhany/chapar/ui/explorer"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/notify"
	"github.com/mirzakhany/chapar/internal/repository"
	"github.com/mirzakhany/chapar/internal/state"
	"github.com/mirzakhany/chapar/ui/fonts"
	"github.com/mirzakhany/chapar/ui/pages/console"
	"github.com/mirzakhany/chapar/ui/pages/environments"
	"github.com/mirzakhany/chapar/ui/pages/requests"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type UI struct {
	Theme  *material.Theme
	window *app.Window

	sideBar *Sidebar
	header  *Header

	consolePage  *console.Console
	notification *widgets.Notification

	environmentsView *environments.View
	requestsView     *requests.View

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

	repo := &repository.Filesystem{}
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

	u.Theme = material.NewTheme()
	u.Theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	// set foreground color
	u.Theme.Palette.Fg = color.NRGBA{R: 0xD7, G: 0xDA, B: 0xDE, A: 0xff}
	// set background color
	u.Theme.Palette.Bg = color.NRGBA{R: 0x20, G: 0x22, B: 0x24, A: 0xff}

	u.Theme.TextSize = unit.Sp(14)
	// console need to be initialized before other pages as its listening for logs
	u.consolePage = console.New()
	u.header = NewHeader(environmentsState)
	u.sideBar = NewSidebar(u.Theme)

	u.environmentsView = environments.NewView(u.Theme)
	envController := environments.NewController(u.environmentsView, repo, environmentsState, explorerController)
	if err := envController.LoadData(); err != nil {
		return nil, err
	}

	u.header.LoadEnvs(environmentsState.GetEnvironments())
	environmentsState.AddEnvironmentChangeListener(func(environment *domain.Environment, source state.Source, action state.Action) {
		u.header.LoadEnvs(environmentsState.GetEnvironments())
	})

	if selectedEnv := environmentsState.GetEnvironment(preferences.Spec.SelectedEnvironment.ID); selectedEnv != nil {
		environmentsState.SetActiveEnvironment(selectedEnv)
		u.header.SetSelectedEnvironment(environmentsState.GetActiveEnvironment())
	}

	u.header.OnSelectedEnvChanged = func(env *domain.Environment) {
		preferences.Spec.SelectedEnvironment.ID = env.MetaData.ID
		preferences.Spec.SelectedEnvironment.Name = env.MetaData.Name
		if err := repo.UpdatePreferences(preferences); err != nil {
			fmt.Println("failed to update preferences: ", err)
		}

		environmentsState.SetActiveEnvironment(env)
	}

	u.requestsView = requests.NewView(u.Theme)
	reqController := requests.NewController(u.requestsView, repo, requestsState, environmentsState, explorerController, restService)
	if err := reqController.LoadData(); err != nil {
		return nil, err
	}

	u.notification = &widgets.Notification{}
	return u, nil
}

func (u *UI) Run() error {
	// expl := explorer.NewExplorer(w)
	// ops are the operations from the UI
	var ops op.Ops

	for {
		switch e := u.window.NextEvent().(type) {
		// this is sent when the application should re-render.
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// set the background color
			paint.ColorOp{Color: u.Theme.Palette.Bg}.Add(&ops)
			paint.PaintOp{}.Add(&ops)
			clip.Rect{Max: gtx.Constraints.Max}.Push(&ops).Pop()

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
	return layout.Stack{Alignment: layout.S}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
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
							switch u.sideBar.SelectedIndex() {
							case 0:
								return u.requestsView.Layout(gtx, u.Theme)
							case 1:
								return u.environmentsView.Layout(gtx, u.Theme)
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
}
