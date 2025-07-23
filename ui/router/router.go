package router

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/console"
	"github.com/chapar-rest/chapar/ui/footer"
	"github.com/chapar-rest/chapar/ui/header"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/sidebar"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Router struct {
	pages   map[any]Page
	current any
	*sidebar.Sidebar
	*component.ModalLayer

	header        *header.Header
	footer        *footer.Footer
	consoleSplit  widgets.SplitView
	consoleLayout *console.Console
}

func New(appVersion string, w *app.Window, envState *state.Environments, workspacesState *state.Workspaces, th *chapartheme.Theme) Router {
	modal := component.NewModal()

	return Router{
		pages:      make(map[any]Page),
		ModalLayer: modal,
		Sidebar:    sidebar.New(),
		consoleSplit: widgets.SplitView{
			Resize: component.Resize{
				Ratio: 0.75,
				Axis:  layout.Vertical,
			},
			BarWidth: unit.Dp(2),
		},
		consoleLayout: console.New(th),
		header:        header.NewHeader(w, envState, workspacesState, th),
		footer: &footer.Footer{
			AppVersion: appVersion,
		},
	}
}

func (r *Router) Register(tag any, p Page) {
	r.pages[tag] = p
	sItem := p.SideBarItem()
	sItem.Tag = tag
	if r.current == any(nil) {
		r.current = tag
	}
	r.Sidebar.AddNavItem(sItem)
}

func (r *Router) SwitchTo(tag any) {
	_, ok := r.pages[tag]
	if !ok {
		return
	}
	r.current = tag
}

func (r *Router) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if r.Sidebar.Changed() {
		r.SwitchTo(r.Sidebar.Current())
	}

	// Paint the background
	paint.Fill(gtx.Ops, th.Palette.Bg)

	layout.Flex{Axis: layout.Vertical, Spacing: 0}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.header.Layout(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if r.consoleLayout.IsVisible() {
				// if console is visible, we use split layout
				return r.layoutWithConsole(gtx, th)
			}

			return r.layoutWithoutConsole(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !r.consoleLayout.IsVisible() {
				return layout.Dimensions{}
			}
			return widgets.Divider(layout.Horizontal, unit.Dp(1)).Layout(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.Divider(layout.Horizontal, unit.Dp(1)).Layout(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.footer.ConsoleClickable.Clicked(gtx) {
				r.consoleLayout.ToggleVisibility()
			}

			if r.footer.NotificationsClickable.Clicked(gtx) {
				notifications.ToggleVisibility()
			}

			return r.footer.Layout(gtx, th)
		}),
	)

	// show the notifications if any
	ops := op.Record(gtx.Ops)
	notifications.Layout(gtx, th)
	defer op.Defer(gtx.Ops, ops.Stop())

	r.ModalLayer.Layout(gtx, th.Material())
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (r *Router) layoutWithoutConsole(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: 0}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(80)
			return r.Sidebar.Layout(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.Divider(layout.Vertical, unit.Dp(1)).Layout(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.pages[r.current].Layout(gtx, th)
		}),
	)
}

func (r *Router) layoutWithConsole(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return r.consoleSplit.Layout(gtx, th,
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.Flex{Axis: layout.Horizontal, Spacing: 0}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(80)
					return r.Sidebar.Layout(gtx, th)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return widgets.Divider(layout.Vertical, unit.Dp(1)).Layout(gtx, th)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return r.pages[r.current].Layout(gtx, th)
				}),
			)
		},
		func(gtx layout.Context) layout.Dimensions {
			return r.consoleLayout.Layout(gtx, th)
		},
	)
}
