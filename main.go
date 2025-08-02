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
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	internal_app "github.com/chapar-rest/chapar/ui/app"
	"github.com/chapar-rest/chapar/ui/chapartheme"
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

		chaparApp, err := internal_app.NewApp(&w, appVersion)
		if err != nil {
			if err := showStartupError(&w, err); err != nil {
				log.Fatal(err)
			}
			os.Exit(1)
		}

		var ops op.Ops
		for {
			switch e := chaparApp.Window.Event().(type) {
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				chaparApp.Layout(gtx, chaparApp.Theme)
				e.Frame(gtx.Ops)
			case app.DestroyEvent:
				os.Exit(0)
			}
		}
	}()

	app.Main()
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
