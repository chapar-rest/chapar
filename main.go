package main

import (
	"flag"
	"image"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/fonts"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/pages/environments"
	"github.com/chapar-rest/chapar/ui/pages/protofiles"
	"github.com/chapar-rest/chapar/ui/pages/requests"
	"github.com/chapar-rest/chapar/ui/pages/settings"
	"github.com/chapar-rest/chapar/ui/pages/workspaces"
	"github.com/chapar-rest/chapar/ui/router"
	"github.com/chapar-rest/chapar/ui/widgets"
)

var (
	appVersion  = ""
	enablePprof = flag.Bool("pprof", false, "enable pprof")
)

func main() {
	flag.Parse()

	if *enablePprof {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	go func() {
		var w app.Window
		w.Option(app.Title("Chapar"), app.Size(unit.Dp(1200), unit.Dp(800)))

		if err := setup(&w); err != nil {
			if err := showStartupError(&w, err); err != nil {
				log.Fatal(err)
			}
			os.Exit(1)
		}

		if err := loop(&w); err != nil {
			if err := showStartupError(&w, err); err != nil {
				log.Fatal(err)
			}
			os.Exit(1)
		}

		//mainUI, err := mainApp.New(&w, serviceVersion)
		//if err != nil {
		//	if err := showStartupError(&w, err); err != nil {
		//		log.Fatal(err)
		//	}
		//	os.Exit(1)
		//}
		//
		//if err := mainUI.Run(); err != nil {
		//	log.Fatal(err)
		//}
		os.Exit(0)
	}()

	app.Main()
}

func loop(w *app.Window) error {
	fontCollection, err := fonts.Prepare()
	if err != nil {
		return err
	}

	appState := prefs.GetAppState()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	th := chapartheme.New(theme, appState.Spec.DarkMode)
	explorerController := explorer.NewExplorer(w)

	// create file storage in user's home directory
	repo, err := repository.NewFilesystemV2(prefs.GetWorkspacePath(), appState.Spec.ActiveWorkspace.Name)
	if err != nil {
		return err
	}

	protoFilesState := state.NewProtoFiles(repo)
	requestsState := state.NewRequests(repo)
	environmentsState := state.NewEnvironments(repo)
	workspacesState, err := state.NewWorkspaces(repo)
	if err != nil {
		return err
	}

	workspaceView := workspaces.NewView()
	protoFilesView := protofiles.NewView()
	settingsView := settings.NewView(w, th)
	environmentsView := environments.NewView(w, th)
	requestsView := requests.NewView(w, th, explorerController)

	workspacesController := workspaces.NewController(workspaceView, workspacesState, repo)
	if err := workspacesController.LoadData(); err != nil {
		return err
	}

	protoFileController := protofiles.NewController(protoFilesView, protoFilesState, repo, explorerController)
	if err := protoFileController.LoadData(); err != nil {
		return err
	}

	grpcService := grpc.NewService(appVersion, requestsState, environmentsState, protoFilesState)
	restService := rest.New(requestsState, environmentsState, appVersion)
	egressService := egress.New(requestsState, environmentsState, restService, grpcService, nil)

	requestsController := requests.NewController(requestsView, repo, requestsState, environmentsState, explorerController, egressService, grpcService)
	if err := requestsController.LoadData(); err != nil {
		return err
	}

	settings.NewController(settingsView)

	r := router.New(appVersion, w, environmentsState, workspacesState, th)
	r.Register("requests", requestsView)
	r.Register("environments", environmentsView)
	r.Register("protofiles", protoFilesView)
	r.Register("workspaces", workspaceView)
	r.Register("settings", settingsView)

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			r.Layout(gtx, th)
			e.Frame(gtx.Ops)
		case app.DestroyEvent:
			return e.Err
		}
	}
}

func setup(w *app.Window) error {
	// init notification system
	notifications.New(w)

	return nil
}

func showStartupError(w *app.Window, err error) error {
	// ops are the operations from the UI
	var (
		ops      op.Ops
		closeBtn = new(widget.Clickable)
	)
	hi, wi := unit.Dp(500), unit.Dp(200)
	w.Option(app.Title("Chapar failed to start"), app.Size(hi, wi), app.MaxSize(hi, wi), app.MinSize(hi, wi))

	for {
		switch e := w.Event().(type) {
		// this is sent when the application should re-render.
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			// render and handle UI.
			// render the error message.
			errorLayout(gtx, closeBtn, err)
			// render and handle the operations from the UI.
			e.Frame(gtx.Ops)
			// this is sent when the application is closed.
		case app.DestroyEvent:
			return e.Err
		}
	}
}

func errorLayout(gtx layout.Context, closeBtn *widget.Clickable, err error) {
	theme := chapartheme.New(material.NewTheme(), true)

	message := err.Error() + "\n\nPlease consider reporting this issue to the Chapar team using github.com/chapar-rest/chapar/issues"

	// set the background color
	macro := op.Record(gtx.Ops)
	rect := image.Rectangle{
		Max: image.Point{
			X: gtx.Constraints.Max.X,
			Y: gtx.Constraints.Max.Y,
		},
	}
	paint.FillShape(gtx.Ops, theme.Palette.Bg, clip.Rect(rect).Op())
	background := macro.Stop()
	background.Add(gtx.Ops)

	if closeBtn.Clicked(gtx) {
		os.Exit(1)
	}

	layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					h := material.H6(theme.Material(), "Chapar failed to start")
					h.Color = theme.ErrorColor
					return h.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					b := material.Body1(theme.Material(), message)
					b.Color = theme.TextColor
					return b.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
						Spacing:   layout.SpaceStart,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := widgets.Button(theme.Material(), closeBtn, nil, widgets.IconPositionStart, "Close")
							btn.Background = chapartheme.White
							btn.Color = chapartheme.Black
							return btn.Layout(gtx, theme)
						}),
					)
				})
			}),
		)
	})
}
