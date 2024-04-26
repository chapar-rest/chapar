package collections

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Collection struct {
	collection *domain.Collection
	Title      *widgets.EditableLabel

	saveButton *widget.Clickable

	prompt *widgets.Prompt

	dataChanged   bool
	onSave        func(id string)
	onDataChanged func(id string, data any)
}

func (c *Collection) SetOnDataChanged(f func(id string, data any)) {
	c.onDataChanged = f
}

func (c *Collection) ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...widgets.Option) {
	c.prompt.Type = modalType
	c.prompt.Title = title
	c.prompt.Content = content
	c.prompt.SetOptions(options...)
	c.prompt.WithoutRememberBool()
	c.prompt.SetOnSubmit(onSubmit)
	c.prompt.Show()
}

func (c *Collection) HidePrompt() {
	c.prompt.Hide()
}

func (c *Collection) SetDataChanged(dirty bool) {
	c.dataChanged = dirty
}

func (c *Collection) SetOnSave(f func(id string)) {
	c.onSave = f
}

func New(collection *domain.Collection) *Collection {
	c := &Collection{
		collection: collection,
		Title:      widgets.NewEditableLabel(collection.MetaData.Name),
		prompt:     widgets.NewPrompt("", "", ""),
		saveButton: new(widget.Clickable),
	}
	c.prompt.WithoutRememberBool()
	return c
}

func (c *Collection) SetOnTitleChanged(f func(string)) {
	c.Title.SetOnChanged(f)
}

func (c *Collection) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if c.onSave != nil {
		keys.OnSaveCommand(gtx, c, func() {
			c.onSave(c.collection.MetaData.ID)
		})
	}

	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return c.prompt.Layout(gtx, theme)
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
									if c.dataChanged && c.onSave != nil {
										if c.saveButton.Clicked(gtx) {
											c.onSave(c.collection.MetaData.ID)
										}
										return widgets.SaveButtonLayout(gtx, theme, c.saveButton)
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
