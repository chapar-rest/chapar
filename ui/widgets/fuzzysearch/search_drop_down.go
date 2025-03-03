package fuzzysearch

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type SearchDropDown struct {
	menuContextArea component.ContextArea
	list            *widget.List
	theme           *chapartheme.Theme

	results []*SearchResult

	recentQueries []string

	input *widgets.TextField

	loaderFunc func() []Item

	OnSelectResult func(result *SearchResult)
}

func NewSearchDropDown(theme *chapartheme.Theme) *SearchDropDown {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

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

	search.SetOnTextChange(c.onSearch)

	return c
}

func (c *SearchDropDown) onSearch(query string) {
	if c.loaderFunc == nil {
		return
	}

	items := c.loaderFunc()
	results := FuzzySearch(items, query, 100)
	if len(results) == 0 {
		return
	}

	for _, item := range results {
		switch item.Item.Kind {
		case domain.KindEnv:
			item.Icon = widgets.MenuIcon
		case domain.KindRequest:
			item.Icon = widgets.SwapHoriz
		case domain.KindWorkspace:
			item.Icon = widgets.WorkspacesIcon
		case domain.KindProtoFile:
			item.Icon = widgets.FileFolderIcon
		case domain.KindCollection:
			item.Icon = widgets.ForwardIcon
		}
	}

	c.results = results
}

func (c *SearchDropDown) SetLoader(fn func() []Item) {
	c.loaderFunc = fn
}

func (c *SearchDropDown) resultItemLayout(gtx layout.Context, theme *chapartheme.Theme, item *SearchResult) layout.Dimensions {
	if item.Clickable.Clicked(gtx) {
		if c.OnSelectResult != nil {
			c.OnSelectResult(item)
		}
	}

	return widgets.Clickable(gtx, &item.Clickable, 0, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(5),
			Bottom: unit.Dp(5),
			Left:   unit.Dp(10),
			Right:  unit.Dp(5),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(18)
					return item.Icon.Layout(gtx, theme.Palette.ContrastFg)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme.Material(), theme.TextSize, item.Item.Title).Layout(gtx)
				}),
			)
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

						return material.List(theme.Material(), c.list).Layout(gtx, len(c.results), func(gtx layout.Context, index int) layout.Dimensions {
							return c.resultItemLayout(gtx, theme, c.results[index])
						})
					})
				})
			})
		}),
	)
}
