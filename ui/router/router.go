package router

import (
	"fmt"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/console"
	"github.com/chapar-rest/chapar/ui/footer"
	"github.com/chapar-rest/chapar/ui/header"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/sidebar"
	"github.com/chapar-rest/chapar/ui/widgets"
)

const (
	RequestsTag     = "requests"
	EnvironmentsTag = "environments"
	ProtoFilesTag   = "protofiles"
	WorkspacesTag   = "workspaces"
	SettingsTag     = "settings"
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

	// dialogs
	MessageDialog modals.Message
}

func New(headerLayout *header.Header, footerLayout *footer.Footer, th *chapartheme.Theme) *Router {
	modal := component.NewModal()

	messageDialog := modals.Message{}

	return &Router{
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
		header:        headerLayout,
		footer:        footerLayout,
		MessageDialog: messageDialog,
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
	r.Update(gtx)

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

	modalOps := op.Record(gtx.Ops)
	r.ModalLayer.Layout(gtx, th.Material())
	defer op.Defer(gtx.Ops, modalOps.Stop())

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

func (r *Router) Clear() {
	r.MessageDialog = modals.Message{}
	r.ModalLayer.Widget = nil

	fmt.Println("Clearing message dialog")
	r.ModalLayer.VisibilityAnimation.Disappear(time.Now())
}

func (r *Router) SetMessageDialog(message modals.Message, th *chapartheme.Theme) {
	r.MessageDialog.Title = message.Title
	r.MessageDialog.Body = message.Body
	r.MessageDialog.Type = message.Type

	// this hack is needed to avoid scrim being disparaged when user is clicking on it
	r.ModalLayer.Scrim.Clickable = widget.Clickable{}
	r.ModalLayer.VisibilityAnimation = component.VisibilityAnimation{
		Duration: 200 * time.Millisecond,
		State:    component.Invisible,
	}

	r.ModalLayer.Widget = func(gtx layout.Context, theme *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions {
		return r.MessageDialog.Layout(gtx, th)
	}

	r.ModalLayer.VisibilityAnimation.Appear(time.Now())
}

func (r *Router) Update(gtx layout.Context) {
	if r.MessageDialog.OKBtn.Clicked(gtx) {
		r.Clear()
	}
}
