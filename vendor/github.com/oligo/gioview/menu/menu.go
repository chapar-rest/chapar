package menu

import (
	"image"
	"image/color"
	"log"

	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var (
	defaultOptionInset = layout.Inset{
		Left:   unit.Dp(20),
		Right:  unit.Dp(20),
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}
)

type Menu struct {
	optionList     widget.List
	options        [][]MenuOption
	optionStates   []*widget.Clickable
	focusedOption  int
	requestDismiss bool
	menuItems      []layout.Widget

	// Background color of the menu. If unset, bg2 of theme will be used.
	Background color.NRGBA
	// Inset applied around the rendered contents of the state's Options field.
	OptionInset layout.Inset
	// Max width of the menu.
	MaxWidth unit.Dp
}

type MenuOption struct {
	Layout    func(gtx C, th *theme.Theme) D
	OnClicked func() error
}

func newMenu(options [][]MenuOption) Menu {
	m := Menu{
		optionList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		options: options,
	}

	return m
}

func (m *Menu) buildMenus(th *theme.Theme) []layout.Widget {
	if len(m.options) <= 0 || (len(m.optionStates) > 0 && len(m.optionStates) == len(m.menuItems)) {
		return nil
	}

	menuItems := make([]layout.Widget, 0)

	idx := 0
	for i, group := range m.options {
		if i != 0 {
			menuItems = append(menuItems, func(gtx C) D {
				return layout.Inset{
					// list scrollbar on the right side has width of 10px or 20px in HiDP system ,
					Left:   unit.Dp(10),
					Bottom: unit.Dp(4),
				}.Layout(gtx, func(gtx C) D {
					return misc.Divider(layout.Horizontal, unit.Dp(1)).Layout(gtx, th)
				})
			})
		}

		for _, opt := range group {
			// closure captured opt
			opt := opt
			if len(m.optionStates) < idx+1 {
				m.optionStates = append(m.optionStates, &widget.Clickable{})
			}

			state := m.optionStates[idx]
			idx++
			menuItems = append(menuItems, func(gtx C) D {
				return m.layoutOption(gtx, th, state, &opt)
			})
		}
	}

	return menuItems
}

func (m *Menu) layout(gtx C, th *theme.Theme, surface func(gtx C, w layout.Widget) D) D {
	if len(m.options) <= 0 {
		return D{}
	}

	macro := op.Record(gtx.Ops)
	gtx.Constraints.Min = gtx.Constraints.Max
	dims := surface(gtx, func(gtx C) D {
		gtx.Constraints.Min = image.Point{}
		return m.layoutOptions(gtx, th)
	})
	menuOps := macro.Stop()

	// Important!!! Otherwise widget below the Menu will not receive pointer events.
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	menuOps.Add(gtx.Ops)
	event.Op(gtx.Ops, m)

	return dims
}

// layoutOptions renders the menu option list.
func (m *Menu) layoutOptions(gtx C, th *theme.Theme) D {
	if len(m.menuItems) <= 0 {
		m.menuItems = m.buildMenus(th)
	}

	maxWidth := gtx.Dp(m.MaxWidth)
	if maxWidth <= 0 {
		var fakeOps op.Ops
		originalOps := gtx.Ops
		gtx.Ops = &fakeOps

		for _, w := range m.menuItems {
			dims := w(gtx)
			if dims.Size.X > maxWidth {
				maxWidth = dims.Size.X
			}
		}
		gtx.Ops = originalOps
	}

	surface := component.Surface(th.Theme)
	surface.Fill = th.Bg

	return surface.Layout(gtx, func(gtx C) D {
		macro := op.Record(gtx.Ops)
		dims := widget.Border{
			Color:        misc.WithAlpha(th.Fg, 0xb6),
			CornerRadius: unit.Dp(4),
			Width:        unit.Dp(0.5),
		}.Layout(gtx, func(gtx C) D {
			return layout.Inset{
				Top:    unit.Dp(8),
				Bottom: unit.Dp(8),
			}.Layout(gtx, func(gtx C) D {
				return material.List(th.Theme, &m.optionList).Layout(gtx, len(m.menuItems), func(gtx C, index int) D {
					gtx.Constraints.Min.X = maxWidth
					gtx.Constraints.Max.X = maxWidth
					return m.menuItems[index](gtx)
				})
			})
		})

		call := macro.Stop()
		defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
		if m.Background == (color.NRGBA{}) {
			paint.ColorOp{Color: th.Bg2}.Add(gtx.Ops)
		} else {
			paint.ColorOp{Color: m.Background}.Add(gtx.Ops)
		}
		paint.PaintOp{}.Add(gtx.Ops)
		call.Add(gtx.Ops)

		return dims
	})

}

func (m *Menu) onDismissed(gtx C) {
	m.optionList.List.ScrollTo(0)
}

func (m *Menu) onActivated(gtx C) {
	// let the menu be focused!
	gtx.Execute(key.FocusCmd{Tag: m})
	m.focusedOption = -1
}

// func (m *Menu) dismiss(onDissmiss func(gtx C)) {
// 	onDissmiss(gtx)
// }

func (m *Menu) update(gtx C) {
	if !gtx.Focused(m) {
		gtx.Execute(key.FocusCmd{Tag: m})
	}

	for {
		e, ok := gtx.Event(
			key.FocusFilter{Target: m},
			key.Filter{Focus: m, Name: key.NameUpArrow},
			key.Filter{Focus: m, Name: key.NameDownArrow},
			key.Filter{Focus: m, Name: key.NameEnter},
			key.Filter{Focus: m, Name: key.NameReturn},
			key.Filter{Focus: m, Name: key.NameEscape},
		)

		if !ok {
			break
		}

		switch e := e.(type) {
		case key.Event:
			// log.Println("menu received key event", e)
			if e.Name == key.NameDownArrow && e.State == key.Release {
				m.focusedOption++
				if m.focusedOption >= len(m.menuItems) {
					m.focusedOption = 0
				}
			}
			if e.Name == key.NameUpArrow && e.State == key.Release {
				m.focusedOption--
				if m.focusedOption < 0 {
					m.focusedOption = len(m.menuItems) - 1
				}
			}
			if (e.Name == key.NameEnter || e.Name == key.NameReturn) && e.State == key.Release {
				log.Println("enter or return key pressed")
				if m.focusedOption >= 0 {
					// simulate a mouse click
					m.optionStates[m.focusedOption].Click()
				}
			}

			if e.Name == key.NameEscape && e.State == key.Release {
				m.requestDismiss = true
				gtx.Execute(op.InvalidateCmd{})
			}
		}
	}

}

func (m *Menu) layoutOption(gtx C, th *theme.Theme, state *widget.Clickable, opt *MenuOption) D {
	if state.Clicked(gtx) {
		opt.OnClicked()
		m.requestDismiss = true
		gtx.Execute(op.InvalidateCmd{})
	}

	if m.OptionInset == (layout.Inset{}) {
		m.OptionInset = defaultOptionInset
	}

	return layout.Inset{
		// list scrollbar on the right side has width of 10px or 20px in HiDP system ,
		Left:   unit.Dp(10),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx C) D {
		return material.Clickable(gtx, state, func(gtx C) D {
			macro := op.Record(gtx.Ops)
			dims := m.OptionInset.Layout(gtx, func(gtx C) D {
				return opt.Layout(gtx, th)
			})
			callOp := macro.Stop()

			defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
			if m.focusedOption >= 0 && m.focusedOption < len(m.optionStates) && m.optionStates[m.focusedOption] == state {
				paint.ColorOp{Color: misc.WithAlpha(th.Fg, th.HoverAlpha)}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
			}

			callOp.Add(gtx.Ops)
			return dims
		})
	})
}
