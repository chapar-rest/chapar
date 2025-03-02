package widgets

import (
	"fmt"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type SearchDropDown struct {
	menuContextArea component.ContextArea
	menu            component.MenuState
	list            *widget.List
	theme           *chapartheme.Theme

	items []*SearchResult

	input *TextField
}

type SearchResult struct {
	Identifier string
	Text       string
	Kind       string

	Clickable widget.Clickable
}

func NewSearchDropDown(theme *chapartheme.Theme, searchResults ...*SearchResult) *SearchDropDown {
	search := NewTextField("", "Search...")
	search.SetIcon(SearchIcon, IconPositionEnd)

	c := &SearchDropDown{
		input: search,
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		theme: theme,
	}

	c.SetSearchResults(searchResults...)

	return c
}

func (c *SearchDropDown) SetSearchResults(results ...*SearchResult) {
	c.items = results
}

func (c *SearchDropDown) resultItemLayout(gtx layout.Context, theme *chapartheme.Theme, item *SearchResult) layout.Dimensions {
	if item.Clickable.Clicked(gtx) {
		fmt.Println("Clicked", item.Text)
	}

	return Clickable(gtx, &item.Clickable, 0, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(5),
			Bottom: unit.Dp(5),
			Left:   unit.Dp(10),
			Right:  unit.Dp(5),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), theme.TextSize, item.Text).Layout(gtx)
		})
	})
}

// Layout the SearchDropDown.
func (c *SearchDropDown) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inputDims := c.input.Layout(gtx, theme)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return inputDims
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return c.menuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				offset := layout.Inset{
					Top:  unit.Dp(float32(inputDims.Size.Y)/gtx.Metric.PxPerDp + 1),
					Left: unit.Dp(0),
				}
				return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					sf := component.Surface(theme.Material())
					sf.Fill = theme.DropDownMenuBgColor
					sf.Elevation = unit.Dp(2)
					return sf.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						gtx.Constraints.Min.Y = gtx.Dp(200)
						gtx.Constraints.Max.Y = gtx.Dp(200)

						return material.List(theme.Material(), c.list).Layout(gtx, len(c.items), func(gtx layout.Context, index int) layout.Dimensions {
							return c.resultItemLayout(gtx, theme, c.items[index])
						})
					})
				})
			})
		}),
	)
}
