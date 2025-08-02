package app

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/console"
	"github.com/chapar-rest/chapar/ui/footer"
	"github.com/chapar-rest/chapar/ui/header"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/pages/environments"
	"github.com/chapar-rest/chapar/ui/pages/protofiles"
	"github.com/chapar-rest/chapar/ui/pages/requests"
	"github.com/chapar-rest/chapar/ui/pages/settings"
	"github.com/chapar-rest/chapar/ui/pages/workspaces"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type BaseLayout struct {
	base *ui.Base
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

	// layouts
	HeaderLayout  *header.Header
	FooterLayout  *footer.Footer
	ConsoleLayout *console.Console

	consoleSplit widgets.SplitView
}

func NewBaseLayout(base *ui.Base) (*BaseLayout, error) {
	// init views
	workspaceView := workspaces.NewView(base)
	protoFilesView := protofiles.NewView(base)
	settingsView := settings.NewView(base)
	environmentsView := environments.NewView(base)
	requestsView := requests.NewView(base)

	// init controllers
	workspacesController := workspaces.NewController(workspaceView, base.WorkspacesState, base.Repository)
	if err := workspacesController.LoadData(); err != nil {
		return nil, err
	}
	protoFileController := protofiles.NewController(protoFilesView, base.ProtoFilesState, base.Repository, base.Explorer)
	if err := protoFileController.LoadData(); err != nil {
		return nil, err
	}

	requestsController := requests.NewController(requestsView, base.Repository, base.RequestsState, base.EnvironmentsState, base.Explorer, base.EgressService, base.GrpcService)
	if err := requestsController.LoadData(); err != nil {
		return nil, err
	}
	// settings view loads its own data from prefs packages.
	settingsController := settings.NewController(settingsView)

	// init environments controller
	environmentsController := environments.NewController(environmentsView, base.Repository, base.EnvironmentsState, base.Explorer)

	headerLayout := header.NewHeader(base.Window, base.EnvironmentsState, base.WorkspacesState, base.Theme)
	footerLayout := footer.New("")
	consoleLayout := console.New(base.Theme)

	return &BaseLayout{
		base:                   base,
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
		HeaderLayout:           headerLayout,
		FooterLayout:           footerLayout,
		ConsoleLayout:          consoleLayout,
		consoleSplit: widgets.SplitView{
			Resize: component.Resize{
				Ratio: 0.75,
				Axis:  layout.Vertical,
			},
			BarWidth: unit.Dp(2),
		},
	}, nil
}

func (b *BaseLayout) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	// Update the navigator state
	b.base.Navigator.Update()

	// Paint the background
	paint.Fill(gtx.Ops, th.Palette.Bg)

	layout.Flex{Axis: layout.Vertical, Spacing: 0}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.HeaderLayout.Layout(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if b.ConsoleLayout.IsVisible() {
				// if console is visible, we use split layout
				return b.layoutWithConsole(gtx, th)
			}

			return b.layoutWithoutConsole(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !b.ConsoleLayout.IsVisible() {
				return layout.Dimensions{}
			}
			return widgets.Divider(layout.Horizontal, unit.Dp(1)).Layout(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.Divider(layout.Horizontal, unit.Dp(1)).Layout(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if b.FooterLayout.ConsoleClickable.Clicked(gtx) {
				b.ConsoleLayout.ToggleVisibility()
			}

			if b.FooterLayout.NotificationsClickable.Clicked(gtx) {
				notifications.ToggleVisibility()
			}

			return b.FooterLayout.Layout(gtx, th)
		}),
	)

	// show the notifications if any
	ops := op.Record(gtx.Ops)
	notifications.Layout(gtx, th)
	defer op.Defer(gtx.Ops, ops.Stop())

	modalOps := op.Record(gtx.Ops)
	b.base.Modal.Layout(gtx, th.Material())
	defer op.Defer(gtx.Ops, modalOps.Stop())

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (b *BaseLayout) layoutWithoutConsole(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: 0}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(80)
			return b.base.Navigator.Sidebar.Layout(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.Divider(layout.Vertical, unit.Dp(1)).Layout(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return b.base.Navigator.Current().Layout(gtx, th)
		}),
	)
}

func (b *BaseLayout) layoutWithConsole(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return b.consoleSplit.Layout(gtx, th,
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.Flex{Axis: layout.Horizontal, Spacing: 0}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(80)
					return b.base.Navigator.Sidebar.Layout(gtx, th)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return widgets.Divider(layout.Vertical, unit.Dp(1)).Layout(gtx, th)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return b.base.Navigator.Current().Layout(gtx, th)
				}),
			)
		},
		func(gtx layout.Context) layout.Dimensions {
			return b.ConsoleLayout.Layout(gtx, th)
		},
	)
}
