package collection

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/bus"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/ui/keys"
	"github.com/mirzakhany/chapar/ui/state"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Collection struct {
	collection *domain.Collection
	title      *widgets.EditableLabel

	dataChanged    bool
	onTitleChanged func(id, title string)
	onDataChanged  func(id string, values []domain.KeyValue)
	saveButton     *widget.Clickable

	prompt *widgets.Prompt
}

func (c *Collection) SetDirty(dirty bool) {
	//TODO implement me
	panic("implement me")
}

func New(collection *domain.Collection) *Collection {
	c := &Collection{
		collection: collection,
		title:      widgets.NewEditableLabel(collection.MetaData.Name),
		prompt:     widgets.NewPrompt("", "", ""),
		saveButton: new(widget.Clickable),
	}
	c.prompt.WithRememberBool()

	c.title.SetOnChanged(func(text string) {
		if c.collection == nil {
			return
		}

		if c.collection.MetaData.Name == text {
			return
		}

		// save changes to the collection
		c.collection.MetaData.Name = text
		if err := loader.UpdateCollection(c.collection); err != nil {
			c.showError(fmt.Sprintf("failed to update environment: %s", err))
			fmt.Println("failed to update collection: ", err)
			return
		}

		if c.onTitleChanged != nil {
			c.onTitleChanged(c.collection.MetaData.ID, text)
			bus.Publish(state.CollectionChanged, nil)
		}
	})

	return c
}

func (c *Collection) SetActiveEnvironment(_ *domain.Environment) {
}

func (c *Collection) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	keys.OnSaveCommand(gtx, c, c.save)

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
									return c.title.Layout(gtx, theme)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if c.dataChanged {
										if c.saveButton.Clicked(gtx) {
											c.save()
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

func (c *Collection) IsDataChanged() bool {
	return c.dataChanged
}

func (c *Collection) SetOnTitleChanged(f func(string, string)) {
	c.onTitleChanged = f
}

func (c *Collection) OnClose() bool {
	if c.dataChanged {
		c.showNotSavedWarning()
		return false
	}

	return true
}

func (c *Collection) showNotSavedWarning() {
	c.prompt.Type = widgets.ModalTypeWarn
	c.prompt.Content = "This collection data is changed, do you wanna save it before closing it?\nHint: you always can save the changes with ctrl+s"
	c.prompt.SetOptions("Yes", "No")
	c.prompt.WithRememberBool()
	c.prompt.SetOnSubmit(c.onPromptSubmit)
	c.prompt.Show()
}

func (c *Collection) onPromptSubmit(selectedOption string, remember bool) {
	if selectedOption == "Yes" {
		c.save()
		c.prompt.Hide()
	}
}

func (c *Collection) save() {
	if c.dataChanged {
		if err := loader.UpdateCollection(c.collection); err != nil {
			c.showError(fmt.Sprintf("failed to update collection: %s", err))
		} else {
			c.dataChanged = false
			bus.Publish(state.CollectionChanged, nil)
		}
	}
}

func (c *Collection) showError(err string) {
	c.prompt.Type = widgets.ModalTypeErr
	c.prompt.Content = err
	c.prompt.SetOptions("I see")
	c.prompt.WithoutRememberBool()
	c.prompt.SetOnSubmit(nil)
	c.prompt.Show()
}
