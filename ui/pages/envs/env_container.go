package envs

import (
	"fmt"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/op/clip"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type envContainer struct {
	items *widgets.KeyValue
	title *widgets.EditableLabel

	env *domain.Environment

	searchBox *widgets.TextField

	deleteButton *widget.Clickable
	saveButton   *widget.Clickable

	prompt *widgets.Prompt

	dataChanged bool

	onTitleChanged func(id, title string)
	onDataChanged  func(id string, values []domain.KeyValue)
}

func newEnvContainer(env *domain.Environment) *envContainer {
	search := widgets.NewTextField("", "Search items")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	c := &envContainer{
		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),
		title:     widgets.NewEditableLabel(""),
		searchBox: search,

		deleteButton: new(widget.Clickable),
		saveButton:   new(widget.Clickable),
		prompt:       widgets.NewPrompt("Save", "This environment value is changed, do you wanna save it before closing it?\nHint: you always can save the changes with ctrl+s", widgets.ModalTypeWarn, "Yes", "No"),
	}
	c.prompt.WithRememberBool()
	c.Load(env)

	c.title.SetOnChanged(func(text string) {
		if c.env == nil {
			return
		}

		if c.env.Meta.Name == text {
			return
		}

		// save changes to the environment
		c.env.Meta.Name = text
		if err := loader.UpdateEnvironment(c.env); err != nil {
			c.showError(fmt.Sprintf("failed to update environment: %s", err))
			return
		}

		if c.onTitleChanged != nil {
			c.onTitleChanged(c.env.Meta.ID, text)
		}
	})

	c.items.SetOnChanged(c.onItemsChange)

	c.searchBox.SetOnTextChange(func(text string) {
		if c.items == nil {
			return
		}

		c.items.Filter(text)
	})

	return c
}

// OnClose is called when the tab is closed
// it returns true if the tab can be closed
func (r *envContainer) OnClose() bool {
	if r.dataChanged {
		r.showNotSavedWarning()
		return false
	}

	return true
}

func (r *envContainer) showError(err string) {
	r.prompt.Type = widgets.ModalTypeErr
	r.prompt.Content = err
	r.prompt.SetOptions("I see")
	r.prompt.WithoutRememberBool()
	r.prompt.SetOnSubmit(nil)
	r.prompt.Show()
}

func (r *envContainer) showNotSavedWarning() {
	r.prompt.Type = widgets.ModalTypeWarn
	r.prompt.Content = "This environment value is changed, do you wanna save it before closing it?\nHint: you always can save the changes with ctrl+s"
	r.prompt.SetOptions("Yes", "No")
	r.prompt.WithRememberBool()
	r.prompt.SetOnSubmit(r.onPromptSubmit)
	r.prompt.Show()
}

func (r *envContainer) onPromptSubmit(selectedOption string, remember bool) {
	if selectedOption == "Yes" {
		r.save()
		r.prompt.Hide()
	}
}

func (r *envContainer) onItemsChange(items []*widgets.KeyValueItem) {
	newEnvValues := make([]domain.KeyValue, 0, len(items))
	for _, vv := range items {
		newEnvValues = append(newEnvValues, domain.KeyValue{
			ID:     vv.Identifier,
			Key:    vv.Key,
			Value:  vv.Value,
			Enable: vv.Active,
		})
	}

	if domain.CompareEnvValues(r.env.Values, newEnvValues) {
		r.dataChanged = false
		return
	}

	r.dataChanged = true
	if r.onDataChanged != nil {
		r.onDataChanged(r.env.Meta.ID, newEnvValues)
	}
}

func (r *envContainer) populateItems() {
	newEnvValues := make([]domain.KeyValue, 0, len(r.items.GetItems()))
	for _, vv := range r.items.GetItems() {
		newEnvValues = append(newEnvValues, domain.KeyValue{
			ID:     vv.Identifier,
			Key:    vv.Key,
			Value:  vv.Value,
			Enable: vv.Active,
		})
	}
	r.env.Values = newEnvValues
}

func (r *envContainer) SetOnTitleChanged(f func(string, string)) {
	r.onTitleChanged = f
}

func (r *envContainer) SetOnDataChanged(f func(string, []domain.KeyValue)) {
	r.onDataChanged = f
}

func (r *envContainer) IsDataChanged() bool {
	return r.dataChanged
}

func (r *envContainer) Load(e *domain.Environment) {
	r.env = e
	r.title.SetText(e.Meta.Name)
	items := make([]*widgets.KeyValueItem, 0, len(e.Values))
	for _, vv := range e.Values {
		items = append(items, widgets.NewKeyValueItem(vv.Key, vv.Value, vv.ID, vv.Enable))
	}

	r.items.SetItems(items)
}

func (r *envContainer) save() {
	if r.dataChanged {
		r.populateItems()
		if err := loader.UpdateEnvironment(r.env); err != nil {
			r.showError(fmt.Sprintf("failed to update environment: %s", err))
		} else {
			r.dataChanged = false
		}
	}
}

func (r *envContainer) saveButtonLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        theme.Palette.ContrastBg,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return layout.Inset{Left: unit.Dp(15)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return widgets.Clickable(gtx, r.saveButton, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(16)
									return widgets.SaveIcon.Layout(gtx, theme.Palette.ContrastFg)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Body1(theme, "Save").Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		})
	})
}

func (r *envContainer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, r)
	for {
		keyEvent, ok := gtx.Event(
			key.Filter{
				Required: key.ModShortcut,
				Name:     "S",
			},
		)
		if !ok {
			break
		}

		if ev, ok := keyEvent.(key.Event); ok {
			if ev.Name == "S" && ev.Modifiers.Contain(key.ModShortcut) && ev.State == key.Press {
				r.save()
			}
		}
	}
	area.Pop()

	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.prompt.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(5),
					Bottom: unit.Dp(15),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return r.title.Layout(gtx, theme)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if r.dataChanged {
										if r.saveButton.Clicked(gtx) {
											r.save()
										}

										return r.saveButtonLayout(gtx, theme)
									} else {
										return layout.Dimensions{}
									}
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(200)
							return r.searchBox.Layout(gtx, theme)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return r.items.WithAddLayout(gtx, "", "Disabled items have no effect on your requests", theme)
			}),
		)
	})
}
