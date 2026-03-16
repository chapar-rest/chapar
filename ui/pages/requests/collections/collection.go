package collections

import (
	"fmt"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Collection struct {
	collection *domain.Collection
	Title      *widgets.EditableLabel

	saveButton *widget.Clickable

	prompt *widgets.Prompt

	Tabs    *widgets.Tabs
	Headers *component.Headers
	Auth    *component.Auth

	notesEditor widget.Editor

	split widgets.SplitView

	dataChanged    bool
	onSave         func(id string)
	onDataChanged  func(id string, data any)
	onTitleChanged func(title string)
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

func New(collection *domain.Collection, theme *chapartheme.Theme) *Collection {
	c := &Collection{
		collection: collection,
		Title:      widgets.NewEditableLabel(collection.MetaData.Name),
		prompt:     widgets.NewPrompt("", "", ""),
		saveButton: new(widget.Clickable),
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Notes"},
			{Title: "Auth"},
			{Title: "Headers"},
		}, nil),
		Headers: component.NewHeaders(collection.Spec.Headers),
		Auth:    component.NewAuth(collection.Spec.Auth, theme),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.6, // 60% left, 40% right
				Axis:  layout.Horizontal,
			},
			BarWidth: unit.Dp(2),
		},
	}

	c.Auth.DropDown.SetOptions(
		widgets.NewDropDownOption("None").WithValue(domain.AuthTypeNone),
		widgets.NewDropDownOption("Basic").WithValue(domain.AuthTypeBasic),
		widgets.NewDropDownOption("Token").WithValue(domain.AuthTypeToken),
		widgets.NewDropDownOption("API Key").WithValue(domain.AuthTypeAPIKey),
	)
	c.Auth.SetAuth(collection.Spec.Auth)

	c.notesEditor.SingleLine = false
	c.notesEditor.SetText(collection.Spec.Notes)
	c.prompt.WithoutRememberBool()
	c.setupHooks()
	return c
}

func (c *Collection) SetOnTitleChanged(f func(string)) {
	c.onTitleChanged = f
}

func (c *Collection) SetTitle(title string) {
	c.Title.SetText(title)
}

func (c *Collection) setupHooks() {
	c.Headers.SetOnChange(func(headers []domain.KeyValue) {
		c.collection.Spec.Headers = headers
		if c.onDataChanged != nil {
			c.onDataChanged(c.collection.MetaData.ID, c.collection)
		}
	})

	c.Auth.SetOnChange(func(auth domain.Auth) {
		c.collection.Spec.Auth = auth
		if c.onDataChanged != nil {
			c.onDataChanged(c.collection.MetaData.ID, c.collection)
		}
	})
}

// getAuthTypeDisplay returns a human-readable auth type string
func (c *Collection) getAuthTypeDisplay() string {
	authType := c.collection.Spec.Auth.Type
	switch authType {
	case domain.AuthTypeNone:
		return "None"
	case domain.AuthTypeBasic:
		return "Basic"
	case domain.AuthTypeToken:
		return "Token"
	case domain.AuthTypeAPIKey:
		return "API Key"
	default:
		if authType == "" {
			return "None"
		}
		return authType
	}
}

// getEnabledHeadersCount returns the count of enabled headers
func (c *Collection) getEnabledHeadersCount() int {
	count := 0
	for _, header := range c.collection.Spec.Headers {
		if header.Enable {
			count++
		}
	}
	return count
}

// pluralize returns the singular or plural form based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (c *Collection) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if c.onSave != nil {
		keys.OnSaveCommand(gtx, c, func() {
			c.onSave(c.collection.MetaData.ID)
		})
	}

	// Handle notes editor changes
	keys.OnEditorChange(gtx, &c.notesEditor, func() {
		c.collection.Spec.Notes = c.notesEditor.Text()
		if c.onDataChanged != nil {
			c.onDataChanged(c.collection.MetaData.ID, c.collection)
		}
	})

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
									dims := c.Title.Layout(gtx, theme)
									if c.Title.Changed() && c.onTitleChanged != nil {
										c.onTitleChanged(c.Title.Text)
									}
									return dims
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
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return c.split.Layout(gtx, theme,
					// Left side: Auth/Headers tabs + Notes
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return c.Tabs.Layout(gtx, theme)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									switch c.Tabs.SelectedTab().Title {
									case "Notes":
										return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												border := widget.Border{
													Color:        theme.BorderColor,
													Width:        unit.Dp(1),
													CornerRadius: unit.Dp(4),
												}
												return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														editor := material.Editor(theme.Material(), &c.notesEditor, "Add notes about this collection...")
														editor.Editor.SingleLine = false
														editor.Editor.Alignment = text.Start
														editor.SelectionColor = theme.TextSelectionColor
														return editor.Layout(gtx)
													})
												})
											}),
										)
									case "Headers":
										return c.Headers.Layout(gtx, theme)
									case "Auth":
										return c.Auth.Layout(gtx, theme)
									default:
										return layout.Dimensions{}
									}
								})
							}),
						)
					},
					// Right side: Overview
					func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									label := material.Label(theme.Material(), unit.Sp(16), "Overview")
									label.Color = theme.TextColor
									label.Font.Weight = font.Bold
									return label.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
								// Requests section
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											icon := widgets.MaterialIcons("swap_horiz", theme)
											icon.Color = theme.TextColor
											icon.TextSize = unit.Sp(18)
											return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, icon.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											requestCount := len(c.collection.Spec.Requests)
											labelText := fmt.Sprintf("%d %s", requestCount, pluralize(requestCount, "request", "requests"))
											label := material.Label(theme.Material(), unit.Sp(14), labelText)
											label.Color = theme.TextColor
											return label.Layout(gtx)
										}),
									)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								// Headers section
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											icon := widgets.MaterialIcons("view_headline", theme)
											icon.Color = theme.TextColor
											icon.TextSize = unit.Sp(18)
											return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, icon.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											headersCount := c.getEnabledHeadersCount()
											labelText := fmt.Sprintf("%d %s", headersCount, pluralize(headersCount, "header", "headers"))
											label := material.Label(theme.Material(), unit.Sp(14), labelText)
											label.Color = theme.TextColor
											return label.Layout(gtx)
										}),
									)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								// Auth section
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											icon := widgets.MaterialIcons("lock", theme)
											icon.Color = theme.TextColor
											icon.TextSize = unit.Sp(18)
											return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, icon.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											authType := c.getAuthTypeDisplay()
											labelText := fmt.Sprintf("Auth method: %s", authType)
											label := material.Label(theme.Material(), unit.Sp(14), labelText)
											label.Color = theme.TextColor
											return label.Layout(gtx)
										}),
									)
								}),
							)
						})
					},
				)
			}),
		)
	})
}
