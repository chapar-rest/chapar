package navi

import (
	"image"

	"github.com/oligo/gioview/menu"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	vertMoreIcon, _ = widget.NewIcon(icons.NavigationMoreVert)
)

type ActionBar struct {
	actions           []view.ViewAction
	overflowState     []widget.Clickable
	overflowedItems   int
	maxVisibleActions int

	// the button that triggers the overflow menu
	overflowBtn  widget.Clickable
	overflowMenu *overflowMenu
}

// overflowMenu holds the state for an overflow menu in an app bar.
type overflowMenu struct {
	// *gvwidget.ModalLayer
	*menu.DropdownMenu
	actionBar    *ActionBar
	lastMenuSize int
}

func (ab *ActionBar) SetActions(actions []view.ViewAction, maxVisibleActions int) {
	ab.actions = actions
	ab.maxVisibleActions = maxVisibleActions
	ab.overflowState = make([]widget.Clickable, len(ab.actions))
	ab.overflowMenu = &overflowMenu{actionBar: ab}
}

func (ab *ActionBar) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	width := 0
	visibleActionItems := 0

	// pre-caculate total width
	fake := gtx
	fake.Ops = &op.Ops{}
	for i, action := range ab.actions {
		state := ab.overflowState[i]
		d := ab.layoutAction(fake, th, &state, action)
		width += d.Size.X
		if width > gtx.Constraints.Max.X {
			break
		}
		visibleActionItems++
	}

	if visibleActionItems <= ab.maxVisibleActions {
		if visibleActionItems != len(ab.actions) {
			// reserve space for the overflow button.
			visibleActionItems--
		}
	} else {
		visibleActionItems = ab.maxVisibleActions
	}

	ab.overflowedItems = len(ab.actions) - visibleActionItems
	var actions []layout.FlexChild
	for i := range ab.actions {
		action := ab.actions[i]
		state := &ab.overflowState[i]
		if i < visibleActionItems {
			actions = append(actions, layout.Rigid(func(gtx C) D {
				return ab.layoutAction(gtx, th, state, action)
			}))
		}
	}

	if ab.overflowedItems > 0 {
		parent := gtx.Constraints
		actions = append(actions, layout.Rigid(func(gtx C) D {
			dims := actionButtonInset.Layout(gtx, func(gtx C) D {
				if ab.overflowBtn.Clicked(gtx) {
					ab.overflowMenu.ToggleVisibility(gtx)
				}
				return misc.IconButton(th, vertMoreIcon, &ab.overflowBtn, "click to show more actions").Layout(gtx)
			})

			gtx.Constraints = parent
			ab.overflowMenu.Layout(gtx, th, dims.Size)
			return dims
		}))
	}

	dims := layout.Flex{Alignment: layout.Middle}.Layout(gtx, actions...)
	// ab.overflowMenu.Layout(gtx, th)
	return dims
}

var actionButtonInset = layout.Inset{
	Top:    unit.Dp(2),
	Bottom: unit.Dp(2),
	Left:   unit.Dp(2),
}

func (ab *ActionBar) layoutAction(gtx layout.Context, th *theme.Theme, state *widget.Clickable, action view.ViewAction) layout.Dimensions {
	return actionButtonInset.Layout(gtx, func(gtx C) D {
		if state.Clicked(gtx) {
			action.OnClicked(gtx)
		}
		return misc.IconButton(th, action.Icon, state, action.Name).Layout(gtx)
	})
}

func (om *overflowMenu) actionForIndex(index int) view.ViewAction {
	offset := len(om.actionBar.actions) - om.actionBar.overflowedItems
	return om.actionBar.actions[offset+index]
}

func (om *overflowMenu) update(gtx C, th *theme.Theme) {
	if om.DropdownMenu != nil {
		om.DropdownMenu.Background = misc.WithAlpha(th.Fg, th.HoverAlpha)
	}
	if om.lastMenuSize == om.actionBar.overflowedItems {
		return
	}

	options := make([]menu.MenuOption, 0)
	for idx := 0; idx < om.actionBar.overflowedItems; idx++ {
		action := om.actionForIndex(idx)
		options = append(options, menu.MenuOption{
			OnClicked: func() error {
				action.OnClicked(gtx)
				return nil
			},
			Layout: func(gtx C, th *theme.Theme) D {
				return layout.Flex{
					Axis:    layout.Horizontal,
					Spacing: layout.SpaceEnd,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return misc.Icon{Icon: action.Icon, Color: th.Fg, Size: unit.Dp(18)}.Layout(gtx, th)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx C) D {
						label := material.Label(th.Theme, th.TextSize, action.Name)
						return label.Layout(gtx)
					}),
				)

			},
		})
	}

	om.DropdownMenu = menu.NewDropdownMenu([][]menu.MenuOption{options})
	om.DropdownMenu.Background = misc.WithAlpha(th.Fg, th.HoverAlpha)
	om.lastMenuSize = om.actionBar.overflowedItems
}

func (om *overflowMenu) Layout(gtx C, th *theme.Theme, pos image.Point) D {
	om.update(gtx, th)
	if om.lastMenuSize == 0 {
		return D{}
	}

	// A tricky way to anchor the floating menu. ContextArea uses gtx.Constraints.Min to layout
	// the menu and returns gtx.Constraints.Min, not the actual menu size! Here we need to get the
	// actual menu size to calculate a position relative to the icon button, so we use a fixed size
	// constraint to let the menu overflow out of the parent constraint.
	gtx.Constraints.Max.Y = 1e6
	gtx.Constraints.Max.X = max(pos.X, gtx.Dp(unit.Dp(125)))
	macro := op.Record(gtx.Ops)
	dims := om.DropdownMenu.Layout(gtx, th)
	call := macro.Stop()

	offset := image.Point{
		X: pos.X - dims.Size.X,
		Y: pos.Y,
	}

	defer op.Offset(offset).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)

	return dims
}
