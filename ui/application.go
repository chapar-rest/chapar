package ui

import (
	"context"
	"sync"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// Application keeps track of all the windows and global state.
type Application struct {
	// Context is used to broadcast application shutdown.
	Context context.Context
	// Shutdown shuts down all windows.
	Shutdown func()
	// active keeps track the open windows, such that application
	// can shut down, when all of them are closed.
	active sync.WaitGroup
}

func NewApplication(ctx context.Context) *Application {
	ctx, cancel := context.WithCancel(ctx)
	return &Application{
		Context:  ctx,
		Shutdown: cancel,
	}
}

// Wait waits for all windows to close.
func (a *Application) Wait() {
	a.active.Wait()
}

// NewWindow creates a new tracked window.
func (a *Application) NewWindow(title string, view View, opts ...app.Option) {
	opts = append(opts, app.Title(title))
	w := &Window{
		App:    a,
		Window: app.NewWindow(opts...),
	}
	a.active.Add(1)
	var errChan = make(chan error)
	go func() {
		defer a.active.Done()
		if err := view.Run(w); err != nil {
			w.App.Shutdown()
			errChan <- err
		}
	}()

	go func() {
		select {
		case <-a.Context.Done():
			w.Perform(system.ActionClose)
		case err := <-errChan:
			w.Perform(system.ActionClose)
			panic(err)
		}
	}()
}

// Window holds window state.
type Window struct {
	App *Application
	*app.Window
}

// View describes .
type View interface {
	// Run handles the window event loop.
	Run(w *Window) error
}

// WidgetView allows to use layout.Widget as a view.
type WidgetView func(gtx layout.Context, th *material.Theme) layout.Dimensions

// Run displays the widget with default handling.
func (view WidgetView) Run(w *Window) error {
	var ops op.Ops

	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	go func() {
		<-w.App.Context.Done()
		w.Perform(system.ActionClose)
	}()
	for {
		switch e := w.NextEvent().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			view(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}
