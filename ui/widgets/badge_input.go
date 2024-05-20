package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type BadgeInput struct {
	Items []*BadgeInputItem

	list *widget.List

	onChange func(values map[string]string)
}

type BadgeInputItem struct {
	Identifier  string
	Value       string
	closeButton widget.Clickable
}

func NewBadgeInput(items ...*BadgeInputItem) *BadgeInput {
	return &BadgeInput{
		Items: items,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Horizontal,
			},
		},
	}
}

func (b *BadgeInput) SetOnChange(f func(values map[string]string)) {
	b.onChange = f
}

func (b *BadgeInput) AddItem(identifier, value string) {
	b.Items = append(b.Items, &BadgeInputItem{
		Identifier: identifier,
		Value:      value,
	})
}

func (b *BadgeInput) GetValues() map[string]string {
	values := make(map[string]string)
	for _, item := range b.Items {
		values[item.Identifier] = item.Value
	}
	return values
}

func (b *BadgeInput) itemLayout(gtx layout.Context, theme *chapartheme.Theme, item *BadgeInputItem) layout.Dimensions {
	return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 8).Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, theme.BadgeBgColor)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(5), Top: unit.Dp(3), Bottom: unit.Dp(3)}.Layout(gtx, material.Label(theme.Material(), theme.TextSize, item.Value).Layout)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(3)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								bkColor := theme.BadgeBgColor
								hoveredColor := Hovered(bkColor)
								if item.closeButton.Hovered() {
									bkColor = hoveredColor
								}
								ib := &IconButton{
									Icon:                 CloseIcon,
									Color:                theme.TextColor,
									BackgroundColor:      bkColor,
									BackgroundColorHover: hoveredColor,
									Size:                 unit.Dp(16),
									Clickable:            &item.closeButton,
								}
								return ib.Layout(gtx, theme)
							})
						}),
					)
				})
			},
		)
	})
}

func (b *BadgeInput) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	for i := range b.Items {
		if b.Items[i].closeButton.Clicked(gtx) {
			b.Items = append(b.Items[:i], b.Items[i+1:]...)

			if b.onChange != nil {
				b.onChange(b.GetValues())
			}

			break
		}
	}

	return b.list.Layout(gtx, len(b.Items), func(gtx layout.Context, i int) layout.Dimensions {
		return b.itemLayout(gtx, theme, b.Items[i])
	})
}
