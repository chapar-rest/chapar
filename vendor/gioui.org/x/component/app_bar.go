package component

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var moreIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationMoreVert)
	return icon
}()

var cancelIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentClear)
	return icon
}()

// VerticalAnchorPosition indicates the anchor position for the content
// of a component. Conventionally, this is use by AppBars and NavDrawers
// to decide how to allocate internal spacing and in which direction to
// animate certain actions.
type VerticalAnchorPosition uint

const (
	Top VerticalAnchorPosition = iota
	Bottom
)

// AppBar implements the material design App Bar documented here:
// https://material.io/components/app-bars-top
//
// TODO(whereswaldon): implement support for RTL layouts
type AppBar struct {
	// init ensures that AppBars constructed using struct literal
	// syntax still have their fields initialized before use.
	init sync.Once

	NavigationButton       widget.Clickable
	NavigationIcon         *widget.Icon
	Title, ContextualTitle string
	// The modal layer is used to lay out the overflow menu. The nav
	// bar is not functional if this field is not supplied.
	*ModalLayer
	// Anchor indicates whether the app bar is anchored at the
	// top or bottom edge of the interface. It defaults to Top, and
	// is used to orient the layout of menus relative to the bar.
	Anchor VerticalAnchorPosition

	normalActions, contextualActions actionGroup
	overflowMenu
	contextualAnim VisibilityAnimation
}

// actionGroup is a logical set of actions that might be displayed
// by an App Bar.
type actionGroup struct {
	actions           []AppBarAction
	actionAnims       []VisibilityAnimation
	overflow          []OverflowAction
	overflowState     []widget.Clickable
	lastOverflowCount int
}

func (a *actionGroup) setActions(actions []AppBarAction, overflows []OverflowAction) {
	a.actions = actions
	a.actionAnims = make([]VisibilityAnimation, len(actions))
	for i := range a.actionAnims {
		a.actionAnims[i].Duration = actionAnimationDuration
	}
	a.overflow = overflows
	a.overflowState = make([]widget.Clickable, len(a.actions)+len(a.overflow))
}

func (a *actionGroup) layout(gtx C, th *material.Theme, overflowBtn *widget.Clickable, overflowDesc string) D {
	overflowedActions := len(a.actions)
	gtx.Constraints.Min.Y = 0
	widthDp := float32(gtx.Constraints.Max.X) / gtx.Metric.PxPerDp
	visibleActionItems := int((widthDp / 48) - 1)
	if visibleActionItems < 0 {
		visibleActionItems = 0
	}
	visibleActionItems = min(visibleActionItems, len(a.actions))
	overflowedActions -= visibleActionItems
	var actions []layout.FlexChild
	for i := range a.actions {
		action := a.actions[i]
		anim := &a.actionAnims[i]
		switch anim.State {
		case Visible:
			if i >= visibleActionItems {
				anim.Disappear(gtx.Now)
			}
		case Invisible:
			if i < visibleActionItems {
				anim.Appear(gtx.Now)
			}
		}
		actions = append(actions, layout.Rigid(func(gtx C) D {
			return action.layout(th.Palette.Bg, th.Palette.Fg, anim, gtx)
		}))
	}
	if len(a.overflow)+overflowedActions > 0 {
		actions = append(actions, layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			btn := material.IconButton(th, overflowBtn, moreIcon, overflowDesc)
			btn.Size = unit.Dp(24)
			btn.Background = th.Palette.Bg
			btn.Color = th.Palette.Fg
			btn.Inset = layout.UniformInset(unit.Dp(6))
			return overflowButtonInset.Layout(gtx, btn.Layout)
		}))
	}
	a.lastOverflowCount = overflowedActions
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, actions...)
}

// overflowMenu holds the state for an overflow menu in an app bar.
type overflowMenu struct {
	*ModalLayer
	list layout.List
	// the button that triggers the overflow menu
	widget.Clickable
	selectedTag interface{}
}

func (o *overflowMenu) updateState(gtx layout.Context, th *material.Theme, barPos VerticalAnchorPosition, actions *actionGroup) {
	o.selectedTag = nil
	if o.Clicked(gtx) && !o.Visible() {
		o.Appear(gtx.Now)
		o.configureOverflow(gtx, th, barPos, actions)
	}
	for i := range actions.overflowState {
		if actions.overflowState[i].Clicked(gtx) {
			o.Disappear(gtx.Now)
			o.selectedTag = o.actionForIndex(i, actions).Tag
		}
	}
}

func (o overflowMenu) overflowLen(actions *actionGroup) int {
	return len(actions.overflow) + actions.lastOverflowCount
}

func (o overflowMenu) actionForIndex(index int, actions *actionGroup) OverflowAction {
	if index < actions.lastOverflowCount {
		return actions.actions[len(actions.actions)-actions.lastOverflowCount+index].OverflowAction
	}
	return actions.overflow[index-actions.lastOverflowCount]
}

// configureOverflow sets the overflowMenu's ModalLayer to display a overflow menu.
func (o *overflowMenu) configureOverflow(gtx C, th *material.Theme, barPos VerticalAnchorPosition, actions *actionGroup) {
	o.ModalLayer.Widget = func(gtx layout.Context, th *material.Theme, anim *VisibilityAnimation) layout.Dimensions {
		width := gtx.Constraints.Max.X / 2
		gtx.Constraints.Min.X = width
		menuMacro := op.Record(gtx.Ops)
		gtx.Constraints.Min.Y = 0
		dims := layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				gtx.Constraints.Min.X = width
				paintRect(gtx, gtx.Constraints.Min, th.Palette.Bg)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx C) D {
				dims := o.list.Layout(gtx, o.overflowLen(actions), func(gtx C, index int) D {
					action := o.actionForIndex(index, actions)
					state := &actions.overflowState[index]
					return material.Clickable(gtx, state, func(gtx C) D {
						gtx.Constraints.Min.X = width
						return layout.Inset{
							Top:    unit.Dp(4),
							Bottom: unit.Dp(4),
							Left:   unit.Dp(8),
						}.Layout(gtx, func(gtx C) D {
							label := material.Label(th, unit.Sp(18), action.Name)
							label.MaxLines = 1
							return label.Layout(gtx)
						})
					})
				})
				return dims
			}),
		)
		menuOp := menuMacro.Stop()
		progress := anim.Revealed(gtx)
		maxWidth, maxHeight := dims.Size.X, dims.Size.Y
		offset := image.Point{
			X: width,
		}
		var rect clip.Rect
		if barPos == Top {
			rect = clip.Rect{
				Max: image.Point{
					X: maxWidth,
					Y: int(float32(dims.Size.Y) * progress),
				},
				Min: image.Point{
					X: maxWidth - int(float32(dims.Size.X)*progress),
					Y: 0,
				},
			}
		} else {
			offset.Y = gtx.Constraints.Max.Y - maxHeight
			rect = clip.Rect{
				Max: image.Point{
					X: maxWidth,
					Y: maxHeight,
				},
				Min: image.Point{
					X: maxWidth - int(float32(dims.Size.X)*progress),
					Y: maxHeight - int(float32(dims.Size.Y)*progress),
				},
			}
		}
		defer op.Offset(offset).Push(gtx.Ops).Pop()
		defer rect.Push(gtx.Ops).Pop()
		menuOp.Add(gtx.Ops)
		return dims
	}
}

// NewAppBar creates and initializes an App Bar.
func NewAppBar(modal *ModalLayer) *AppBar {
	ab := &AppBar{
		overflowMenu: overflowMenu{
			ModalLayer: modal,
		},
	}
	ab.initialize()
	return ab
}

func (a *AppBar) initialize() {
	a.init.Do(func() {
		a.overflowMenu.list.Axis = layout.Vertical
		a.contextualAnim.State = Invisible
		a.contextualAnim.Duration = contextualAnimationDuration
	})
}

// AppBarAction configures an action in the App Bar's action items.
// The embedded OverflowAction provides the action information for
// when this item disappears into the overflow menu.
type AppBarAction struct {
	OverflowAction
	Layout func(gtx layout.Context, bg, fg color.NRGBA) layout.Dimensions
}

// SimpleIconAction configures an AppBarAction that functions as a simple
// IconButton. To receive events from the button, use the standard methods
// on the provided state parameter.
func SimpleIconAction(state *widget.Clickable, icon *widget.Icon, overflow OverflowAction) AppBarAction {
	a := AppBarAction{
		OverflowAction: overflow,
		Layout: func(gtx C, bg, fg color.NRGBA) D {
			btn := SimpleIconButton(bg, fg, state, icon)
			return btn.Layout(gtx)
		},
	}
	return a
}

// SimpleIconButton creates an IconButtonStyle that is pre-configured to
// be the right size for use as an AppBarAction
func SimpleIconButton(bg, fg color.NRGBA, state *widget.Clickable, icon *widget.Icon) material.IconButtonStyle {
	return material.IconButtonStyle{
		Background: bg,
		Color:      fg,
		Icon:       icon,
		Size:       unit.Dp(24),
		Inset:      layout.UniformInset(unit.Dp(12)),
		Button:     state,
	}
}

const (
	actionAnimationDuration     = time.Millisecond * 250
	contextualAnimationDuration = time.Millisecond * 250
)

var actionButtonInset = layout.Inset{
	Top:    unit.Dp(4),
	Bottom: unit.Dp(4),
}

func (a AppBarAction) layout(bg, fg color.NRGBA, anim *VisibilityAnimation, gtx layout.Context) layout.Dimensions {
	if !anim.Visible() {
		return layout.Dimensions{}
	}
	animating := anim.Animating()
	var macro op.MacroOp
	if animating {
		macro = op.Record(gtx.Ops)
	}
	if !animating {
		return a.Layout(gtx, bg, fg)
	}
	dims := actionButtonInset.Layout(gtx, func(gtx C) D {
		return a.Layout(gtx, bg, fg)
	})
	btnOp := macro.Stop()
	progress := anim.Revealed(gtx)
	dims.Size.X = int(progress * float32(dims.Size.X))

	// ensure this clip transformation stays local to this function
	defer clip.Rect{
		Max: dims.Size,
	}.Push(gtx.Ops).Pop()
	btnOp.Add(gtx.Ops)
	return dims
}

var overflowButtonInset = layout.Inset{
	Top:    unit.Dp(10),
	Bottom: unit.Dp(10),
}

// OverflowAction holds information about an action available in an overflow menu
type OverflowAction struct {
	Name string
	Tag  interface{}
}

func Interpolate(a, b color.NRGBA, progress float32) color.NRGBA {
	var out color.NRGBA
	out.R = uint8(int16(a.R) - int16(float32(int16(a.R)-int16(b.R))*progress))
	out.G = uint8(int16(a.G) - int16(float32(int16(a.G)-int16(b.G))*progress))
	out.B = uint8(int16(a.B) - int16(float32(int16(a.B)-int16(b.B))*progress))
	out.A = uint8(int16(a.A) - int16(float32(int16(a.A)-int16(b.A))*progress))
	return out
}

// SwapGrounds swaps the foreground and background colors
// of both the contrast and non-constrast colors.
//
// Bg <-> Fg
// ContrastBg <-> ContrastFg
func SwapGrounds(p material.Palette) material.Palette {
	out := p
	out.Fg, out.Bg = out.Bg, out.Fg
	out.ContrastFg, out.ContrastBg = out.ContrastBg, out.ContrastFg
	return out
}

// SwapPairs swaps the contrast and non-constrast colors.
//
// Bg <-> ContrastBg
// Fg <-> ContrastFg
func SwapPairs(p material.Palette) material.Palette {
	out := p
	out.Bg, out.ContrastBg = out.ContrastBg, out.Bg
	out.Fg, out.ContrastFg = out.ContrastFg, out.Fg
	return out
}

// Layout renders the app bar. It will span all available horizontal
// space (gtx.Constraints.Max.X), but has a fixed height. The navDesc
// is an accessibility description for the navigation icon button, and
// the overflowDesc is an accessibility description for the overflow
// action button.
func (a *AppBar) Layout(gtx layout.Context, theme *material.Theme, navDesc, overflowDesc string) layout.Dimensions {
	a.initialize()
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(56))
	th := *theme

	normalBg := th.Palette.ContrastBg
	if a.contextualAnim.Visible() {
		// switch the foreground and background colors
		th.Palette = SwapGrounds(th.Palette)
	} else {
		// switch the contrast and main colors
		th.Palette = SwapPairs(th.Palette)
	}

	actionSet := &a.normalActions
	if a.contextualAnim.Visible() {
		if a.contextualAnim.Animating() {
			th.Palette.Bg = Interpolate(normalBg, th.Palette.Bg, a.contextualAnim.Revealed(gtx))
		}
		actionSet = &a.contextualActions
	}
	paintRect(gtx, gtx.Constraints.Max, th.Palette.Bg)
	overflowTh := th.WithPalette(SwapGrounds(th.Palette))
	a.overflowMenu.updateState(gtx, &overflowTh, a.Anchor, actionSet)

	layout.Flex{
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if a.NavigationIcon == nil {
				return layout.Dimensions{}
			}
			icon := a.NavigationIcon
			if a.contextualAnim.Visible() {
				icon = cancelIcon
			}
			button := material.IconButton(&th, &a.NavigationButton, icon, navDesc)
			button.Size = unit.Dp(24)
			button.Background = th.Palette.Bg
			button.Color = th.Fg
			button.Inset = layout.UniformInset(unit.Dp(16))
			return button.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Left: unit.Dp(16)}.Layout(gtx, func(gtx C) D {
				titleText := a.Title
				if a.contextualAnim.Visible() {
					titleText = a.ContextualTitle
				}
				title := material.Body1(&th, titleText)
				title.TextSize = unit.Sp(18)
				return title.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return layout.E.Layout(gtx, func(gtx C) D {
				return actionSet.layout(gtx, &th, &a.overflowMenu.Clickable, overflowDesc)
			})
		}),
	)
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AppBarEvent
type AppBarEvent interface {
	AppBarEvent()
}

// AppBarNavigationClicked indicates that the navigation icon was clicked
// during the last frame.
type AppBarNavigationClicked struct{}

func (a AppBarNavigationClicked) AppBarEvent() {}

func (a AppBarNavigationClicked) String() string {
	return "clicked app bar navigation button"
}

// AppBarContextMenuDismissed indicates that the app bar context menu was
// dismissed during the last frame.
type AppBarContextMenuDismissed struct{}

func (a AppBarContextMenuDismissed) AppBarEvent() {}

func (a AppBarContextMenuDismissed) String() string {
	return "dismissed app bar context menu"
}

// AppBarOverflowActionClicked indicates that an action in the app bar overflow
// menu was clicked during the last frame.
type AppBarOverflowActionClicked struct {
	Tag interface{}
}

func (a AppBarOverflowActionClicked) AppBarEvent() {}

func (a AppBarOverflowActionClicked) String() string {
	return fmt.Sprintf("clicked app bar overflow action with tag %v", a.Tag)
}

// Events returns a slice of all AppBarActions to occur since the last frame.
func (a *AppBar) Events(gtx layout.Context) []AppBarEvent {
	var out []AppBarEvent
	if clicked := a.NavigationButton.Clicked(gtx); clicked && a.contextualAnim.Visible() {
		a.contextualAnim.Disappear(gtx.Now)
		out = append(out, AppBarContextMenuDismissed{})
	} else if clicked {
		out = append(out, AppBarNavigationClicked{})
	}
	if a.overflowMenu.selectedTag != nil {
		out = append(out, AppBarOverflowActionClicked{Tag: a.overflowMenu.selectedTag})
	}
	return out
}

// SetActions configures the set of actions available in the
// action item area of the app bar. They will be displayed
// in the order provided (from left to right) and will
// collapse into the overflow menu from right to left. The
// provided OverflowActions will always be in the overflow
// menu in the order provided.
func (a *AppBar) SetActions(actions []AppBarAction, overflows []OverflowAction) {
	a.normalActions.setActions(actions, overflows)
}

// SetContextualActions configures the actions that should be available in
// the next Contextual mode that this action bar enters.
func (a *AppBar) SetContextualActions(actions []AppBarAction, overflows []OverflowAction) {
	a.contextualActions.setActions(actions, overflows)
}

// StartContextual causes the AppBar to transform into a contextual
// App Bar with a different set of actions than normal. If the App Bar
// is already in contextual mode, this has no effect. The title will
// be used as the contextual app bar title.
func (a *AppBar) StartContextual(when time.Time, title string) {
	if !a.contextualAnim.Visible() {
		a.contextualAnim.Appear(when)
		a.ContextualTitle = title
	}
}

// StopContextual causes the AppBar to stop showing a contextual menu
// if one is currently being displayed.
func (a *AppBar) StopContextual(when time.Time) {
	if a.contextualAnim.Visible() {
		a.contextualAnim.Disappear(when)
	}
}

// ToggleContextual switches between contextual an noncontextual mode.
// If it transitions to contextual mode, the provided title is used.
func (a *AppBar) ToggleContextual(when time.Time, title string) {
	if !a.contextualAnim.Visible() {
		a.StartContextual(when, title)
	} else {
		a.StopContextual(when)
	}
}

// CloseOverflowMenu requests that the overflow menu be closed if it is
// open.
func (a *AppBar) CloseOverflowMenu(when time.Time) {
	if a.overflowMenu.Visible() {
		a.overflowMenu.Disappear(when)
	}
}
