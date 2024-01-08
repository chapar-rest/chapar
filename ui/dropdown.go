package ui

//
//type Option struct {
//	Name     string
//	CallBack func()
//
//	IsDivider        bool
//	IsNoneSelectable bool
//}
//
//func NewOption(name string, callBack func()) Option {
//	return Option{
//		Name:     name,
//		CallBack: callBack,
//	}
//}
//
//func NewDivider() Option {
//	return Option{
//		IsDivider: true,
//	}
//}
//
//func NewNoneSelectableOption(name string, callback func()) Option {
//	return Option{
//		Name:             name,
//		IsNoneSelectable: true,
//		CallBack:         callback,
//	}
//}
//
//type Dropdown struct {
//	theme         *material.Theme
//	Options       []Option
//	SelectedIndex int
//	Button        widget.Clickable
//	isOpen        bool
//
//	optionsClickable []widget.Clickable
//
//	menuContextArea component.ContextArea
//	envMenu         component.MenuState
//
//	dropDownIcon *widget.Icon
//
//	menuInit bool
//}
//
//func NewDropdown(th *material.Theme, options []Option) *Dropdown {
//	d := &Dropdown{
//		theme:            th,
//		Options:          options,
//		optionsClickable: make([]widget.Clickable, len(options)),
//		menuContextArea: component.ContextArea{
//			Activation:       pointer.ButtonPrimary,
//			AbsolutePosition: true,
//		},
//		isOpen: false, // it's a bug, if this is false, the dropdown will not show up on the first click
//	}
//
//	d.dropDownIcon, _ = widget.NewIcon(icons.NavigationArrowDropDown)
//
//	for i := range d.Options {
//		i := i
//		cl := &d.optionsClickable[i]
//
//		d.envMenu.Options = append(d.envMenu.Options, func(gtx C) D {
//			if options[i].IsDivider {
//				return component.Divider(th).Layout(gtx)
//			}
//
//			for cl.Clicked(gtx) {
//				d.SelectedIndex = i
//				d.isOpen = false
//
//				if options[i].CallBack != nil {
//					options[i].CallBack()
//				}
//			}
//			return component.MenuItem(th, cl, options[i].Name).Layout(gtx)
//		})
//	}
//
//	return d
//}
//
//func (d *Dropdown) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
//	btnSize := 100
//	for _, o := range d.envMenu.Options {
//		if o(gtx).Size.X > btnSize {
//			btnSize = o(gtx).Size.X
//		}
//	}
//
//	gtx.Constraints.Min.X = btnSize
//	btn := material.Button(th, &d.Button, d.Options[d.SelectedIndex].Name)
//	btn.Background = th.Bg
//	btn.Inset = layout.Inset{
//		Top:    10,
//		Bottom: 10,
//		Left:   0,
//		Right:  5,
//	}
//	// Lay out the button and capture its dimensions.
//	btnDims := btn.Layout(gtx)
//
//	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
//		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
//
//			return btnDims
//		}),
//		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
//			return d.menuContextArea.Layout(gtx, func(gtx C) D {
//				gtx.Constraints.Min = image.Point{}
//				offset := layout.Inset{
//					Top: unit.Dp(float32(btnDims.Size.Y)/gtx.Metric.PxPerDp + 5),
//				}
//
//				return offset.Layout(gtx, func(gtx C) D {
//					return component.Menu(th, &d.envMenu).Layout(gtx)
//				})
//
//			})
//		}),
//	)
//}
