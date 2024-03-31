package environments

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/converter"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type container struct {
	// env container
	Items       *widgets.KeyValue
	Identifier  string
	Title       *widgets.EditableLabel
	SearchBox   *widgets.TextField
	SaveButton  widget.Clickable
	Prompt      *widgets.Prompt
	DataChanged bool
}

func newContainer(id, name string, items []domain.KeyValue) *container {
	search := widgets.NewTextField("", "Search items")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	c := &container{
		Identifier: id,
		Items:      widgets.NewKeyValue(converter.WidgetItemsFromKeyValue(items)...),
		Title:      widgets.NewEditableLabel(name),
		SearchBox:  search,
		SaveButton: widget.Clickable{},
		Prompt:     widgets.NewPrompt("Save", "", widgets.ModalTypeWarn),
	}
	c.Prompt.WithoutRememberBool()
	return c
}

func (c *container) Layout(gtx layout.Context, theme *material.Theme, selectedID string) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return c.Prompt.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(5),
					Bottom: unit.Dp(15),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return c.Title.Layout(gtx, theme)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if c.DataChanged {
										return widgets.SaveButtonLayout(gtx, theme, &c.SaveButton)
									} else {
										return layout.Dimensions{}
									}
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(200)
							return c.SearchBox.Layout(gtx, theme)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return c.Items.WithAddLayout(gtx, "", "Disabled items have no effect on your requests", theme)
			}),
		)
	})
}
