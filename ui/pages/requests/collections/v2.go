package collections

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type CollectionsV2 struct {
	*ui.Base

	collection  *domain.Collection
	dataChanged bool

	title *widgets.EditableLabel

	saveButton widget.Clickable
}

func NewV2(base *ui.Base, collection *domain.Collection) *CollectionsV2 {
	col := *collection // make a copy

	return &CollectionsV2{
		Base:       base,
		collection: &col,
		title:      widgets.NewEditableLabel(collection.MetaData.Name),
		saveButton: widget.Clickable{},
	}
}

func (c *CollectionsV2) DataChanged() bool {
	return c.dataChanged
}

func (c *CollectionsV2) Close() error {
	return nil
}

func (c *CollectionsV2) update() {
	if c.title.Text != c.collection.MetaData.Name {
		c.dataChanged = true
	}
}

func (c *CollectionsV2) save() {
	c.collection.MetaData.Name = c.title.Text
	c.dataChanged = false
	if err := c.Repository.UpdateCollection(c.collection); err != nil {
		c.ShowError(err)
	}
	c.Window.Invalidate()
}

func (c *CollectionsV2) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	c.update()

	keys.OnSaveCommand(gtx, c, c.save)

	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(5),
					Bottom: unit.Dp(15),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return c.title.Layout(gtx, th)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if c.dataChanged {
										if c.saveButton.Clicked(gtx) {
											c.save()
										}
										return widgets.SaveButtonLayout(gtx, th, &c.saveButton)
									} else {
										return layout.Dimensions{}
									}
								}),
							)
						}),
					)
				})
			}),
		)
	})
}
