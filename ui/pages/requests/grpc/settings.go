package grpc

import (
	"strconv"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

const (
	ItemTypeText    = "text"
	ItemTypeBool    = "bool"
	ItemTypeLNumber = "number"
)

type Settings struct {
	Items []*Item

	list *widget.List
}

func NewSettings(Items []*Item) *Settings {
	return &Settings{
		Items: Items,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

type Item struct {
	Title       string
	Description string
	Type        string
	Value       any

	boolState *widget.Bool
	editor    *widget.Editor
}

func NewTextItem(title, description string, value string) *Item {
	i := &Item{
		Title:       title,
		Description: description,
		Type:        ItemTypeText,
		Value:       value,
		editor:      &widget.Editor{SingleLine: true, Alignment: text.Middle},
	}
	i.editor.SetText(value)
	return i
}

func NewBoolItem(title, description string, value bool) *Item {
	return &Item{
		Title:       title,
		Description: description,
		Type:        ItemTypeBool,
		Value:       value,
		boolState:   new(widget.Bool),
	}
}

func NewNumberItem(title, description string, value int) *Item {
	i := &Item{
		Title:       title,
		Description: description,
		Type:        ItemTypeLNumber,
		Value:       value,
		editor:      &widget.Editor{SingleLine: true, Alignment: text.Middle},
	}
	i.editor.SetText(strconv.Itoa(value))
	return i
}

func (i *Item) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(15)}

	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Label(theme.Material(), unit.Sp(14), i.Title).Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(theme.Material(), unit.Sp(12), i.Description)
						lb.Color = widgets.Disabled(theme.TextColor)
						return lb.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					switch i.Type {
					case ItemTypeText:
						return i.editorLayout(gtx, theme)
					case ItemTypeBool:
						return i.switchLayout(gtx, theme)
					case ItemTypeLNumber:
						return i.editorLayout(gtx, theme)
					default:
						return layout.Dimensions{}
					}
				})
			}),
		)
	})
}

func (i *Item) switchLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	desc := "OFF"
	if i.boolState.Value {
		desc = "ON"
	}
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			s := material.Switch(theme.Material(), i.boolState, "")
			s.Color.Enabled = theme.SwitchBgColor
			s.Color.Disabled = theme.Palette.Fg
			return s.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), unit.Sp(12), desc).Layout(gtx)
		}),
	)
}

func (i *Item) editorLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(50)
	border := widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			editor := material.Editor(theme.Material(), i.editor, "")
			editor.TextSize = unit.Sp(14)
			return editor.Layout(gtx)
		})
	})
}

func (s *Settings) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(15), Right: unit.Dp(20)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.list.Layout(gtx, len(s.Items), func(gtx layout.Context, i int) layout.Dimensions {
			return s.Items[i].Layout(gtx, theme)
		})
	})
}
