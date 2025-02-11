package component

import (
	"fmt"
	"strconv"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Variables struct {
	theme *chapartheme.Theme

	// TODO define request type as enum
	// it ca nbe either http or grpc
	requestType string

	Items     []*Variable
	addButton *widgets.IconButton
	list      *widget.List

	onChanged func(values []domain.Variable)

	previewTitle string
	previewValue string

	responseDetail *domain.ResponseDetail
}

type Variable struct {
	Identifier        string
	TargetEnvVariable string              // The environment variable to set
	From              domain.VariableFrom // Source: "body", "header", "cookie"
	SourceKey         string              // For "header" or "cookie", specify the key name
	OnStatusCode      int                 // Trigger on a specific status code
	JsonPath          string              // JSONPath for extracting value (for "body")
	Enable            bool                // Enable or disable this variable

	targetEnvEditor    *widget.Editor
	fromDropDown       *widgets.DropDown
	sourceKeyEditor    *widget.Editor
	onStatusCodeEditor *widgets.NumericEditor
	jsonPathCodeEditor *widget.Editor

	enableBool   *widget.Bool
	deleteButton widget.Clickable
}

func NewVariables(theme *chapartheme.Theme, requestType string, items ...*Variable) *Variables {
	f := &Variables{
		theme:       theme,
		requestType: requestType,
		addButton: &widgets.IconButton{
			Icon:      widgets.PlusIcon,
			Size:      unit.Dp(20),
			Clickable: &widget.Clickable{},
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}

	for _, item := range items {
		f.addItem(item)
	}

	f.addButton.OnClick = func() {
		v := NewVariable()
		if f.requestType == domain.RequestTypeGRPC {
			v.OnStatusCode = 0
		}

		f.addItem(NewVariable())
		if f.onChanged != nil {
			f.onChanged(f.GetValues())
		}
	}

	return f
}

func (f *Variables) SetOnChanged(fn func(values []domain.Variable)) {
	f.onChanged = fn
}

func (f *Variables) SetResponseDetail(resp *domain.ResponseDetail) {
	f.responseDetail = resp
}

func (f *Variables) GetValues() []domain.Variable {
	values := make([]domain.Variable, 0, len(f.Items))
	for _, item := range f.Items {
		values = append(values, domain.Variable{
			ID:                item.Identifier,
			TargetEnvVariable: item.TargetEnvVariable,
			From:              item.From,
			SourceKey:         item.SourceKey,
			OnStatusCode:      item.OnStatusCode,
			JsonPath:          item.JsonPath,
			Enable:            item.Enable,
		})
	}
	return values
}

func (f *Variables) SetValues(values []domain.Variable) {
	f.Items = make([]*Variable, 0, len(values))
	for _, item := range values {
		f.addItem(&Variable{
			Identifier:        item.ID,
			TargetEnvVariable: item.TargetEnvVariable,
			From:              item.From,
			SourceKey:         item.SourceKey,
			OnStatusCode:      item.OnStatusCode,
			JsonPath:          item.JsonPath,
			Enable:            item.Enable,
		})
	}
}

func (f *Variables) addItem(item *Variable) {
	if f.requestType == domain.RequestTypeHTTP {
		item.fromDropDown = widgets.NewDropDownWithoutBorder(
			f.theme,
			widgets.NewDropDownOption("Body").WithIdentifier("body").WithValue("body"),
			widgets.NewDropDownOption("Header").WithIdentifier("header").WithValue("header"),
			widgets.NewDropDownOption("Cookie").WithIdentifier("cookie").WithValue("cookie"),
		)
	} else if f.requestType == domain.RequestTypeGRPC {
		item.fromDropDown = widgets.NewDropDownWithoutBorder(
			f.theme,
			widgets.NewDropDownOption("Body").WithIdentifier("body").WithValue("body"),
			widgets.NewDropDownOption("Meta").WithIdentifier("metadata").WithValue("metadata"),
			widgets.NewDropDownOption("Trailers").WithIdentifier(domain.VariableFromTrailers.String()).WithValue(domain.VariableFromTrailers.String()),
		)
	}

	item.fromDropDown.SetSelectedByValue(item.From.String())
	item.fromDropDown.MinWidth = unit.Dp(60)
	item.fromDropDown.MaxWidth = unit.Dp(80)

	item.targetEnvEditor = &widget.Editor{SingleLine: true}
	item.targetEnvEditor.SetText(item.TargetEnvVariable)

	item.sourceKeyEditor = &widget.Editor{SingleLine: true}
	item.sourceKeyEditor.SetText(item.SourceKey)

	item.onStatusCodeEditor = &widgets.NumericEditor{Editor: widget.Editor{SingleLine: true}}
	item.onStatusCodeEditor.Editor.SetText(strconv.Itoa(item.OnStatusCode))

	item.enableBool = new(widget.Bool)
	item.enableBool.Value = item.Enable

	item.fromDropDown.SetOnChanged(func(selected string) {
		item.From = domain.VariableFrom(selected)
		f.triggerChanged()
	})

	item.jsonPathCodeEditor = &widget.Editor{SingleLine: true}
	item.jsonPathCodeEditor.SetText(item.JsonPath)

	f.Items = append(f.Items, item)
}

func NewVariable() *Variable {
	return &Variable{
		Identifier:        uuid.NewString(),
		TargetEnvVariable: "",
		From:              domain.VariableFromBody,
		SourceKey:         "",
		OnStatusCode:      200,
		JsonPath:          "",
		Enable:            true,
	}
}

func (f *Variables) itemLayout(gtx layout.Context, theme *chapartheme.Theme, item *Variable) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, f.itemLayouts(gtx, theme, item)...)
}

func (f *Variables) triggerChanged() {
	if f.onChanged != nil {
		f.onChanged(f.GetValues())
	}
}

func (f *Variables) itemLayouts(gtx layout.Context, theme *chapartheme.Theme, item *Variable) []layout.FlexChild {
	f.update(gtx, item)

	items := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ch := widgets.CheckBox(theme.Material(), item.enableBool, "")
			ch.IconColor = theme.CheckBoxColor
			return ch.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(1), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(100))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(100))
				editor := material.Editor(theme.Material(), item.targetEnvEditor, "Target Key")
				editor.SelectionColor = theme.TextSelectionColor
				return editor.Layout(gtx)
			})
		}),
		widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return item.fromDropDown.Layout(gtx, theme)
		}),
		widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(40))
				return item.onStatusCodeEditor.Layout(gtx, theme)
			})
		}),
		widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
	}

	itemType := item.fromDropDown.GetSelected().Identifier
	if itemType == string(domain.VariableFromBody) {
		items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				editor := material.Editor(theme.Material(), item.jsonPathCodeEditor, "e.g. $.data[0].name")
				editor.SelectionColor = theme.TextSelectionColor
				return editor.Layout(gtx)
			})
		}))
	} else {
		items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				editor := material.Editor(theme.Material(), item.sourceKeyEditor, "Source Key")
				editor.SelectionColor = theme.TextSelectionColor
				return editor.Layout(gtx)
			})
		}))
	}

	items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		ib := widgets.IconButton{
			Icon:      widgets.DeleteIcon,
			Size:      unit.Dp(18),
			Color:     theme.TextColor,
			Clickable: &item.deleteButton,
		}
		return ib.Layout(gtx, theme)
	}))

	return items
}

func (f *Variables) headerLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(128))
				return material.Label(theme.Material(), theme.TextSize, "Target Key").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.DrawLine(gtx, theme.TableBorderColor, unit.Dp(35), unit.Dp(1))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
				return material.Label(theme.Material(), theme.TextSize, "From").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.DrawLine(gtx, theme.TableBorderColor, unit.Dp(35), unit.Dp(1))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(40))
				return material.Label(theme.Material(), theme.TextSize, "Status").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widgets.DrawLine(gtx, theme.TableBorderColor, unit.Dp(35), unit.Dp(1))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme.Material(), theme.TextSize, "Source Key/JSON Path").Layout(gtx)
			})
		}),
	)
}

func (f *Variables) layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return f.headerLayout(gtx, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(f.Items) == 0 {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, material.Label(theme.Material(), unit.Sp(14), "No items").Layout)
			}
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.List(theme.Material(), f.list).Layout(gtx, len(f.Items), func(gtx layout.Context, i int) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return f.itemLayout(gtx, theme, f.Items[i])
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// only if it's not the last item
							if i == len(f.Items)-1 {
								return layout.Dimensions{}
							}
							return widgets.DrawLine(gtx, theme.TableBorderColor, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X))
						}),
					)
				})
			})
		}),
	)
}

func (f *Variables) Layout(gtx layout.Context, title, hint string, theme *chapartheme.Theme) layout.Dimensions {
	for i, field := range f.Items {
		if field.deleteButton.Clicked(gtx) {
			f.Items = append(f.Items[:i], f.Items[i+1:]...)
			f.triggerChanged()
		}
	}

	border := widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}

	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	prevInset := layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(4), Bottom: unit.Dp(4)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme.Material(), theme.TextSize, title).Layout(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Left:  unit.Dp(10),
							Right: unit.Dp(10),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Label(theme.Material(), unit.Sp(10), hint).Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top:    0,
							Bottom: unit.Dp(10),
							Left:   0,
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							f.addButton.BackgroundColor = theme.Palette.Bg
							f.addButton.Color = theme.TextColor
							return f.addButton.Layout(gtx, theme)
						})
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return f.layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:   unit.Dp(10),
					Right: unit.Dp(0),
					Left:  unit.Dp(0),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return prevInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return material.Label(theme.Material(), theme.TextSize, "Preview: "+f.previewTitle).Layout(gtx)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return prevInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return material.Label(theme.Material(), theme.TextSize, f.previewValue).Layout(gtx)
								})
							}),
						)
					})
				})
			}),
		)
	})
}

func (f *Variables) update(gtx layout.Context, item *Variable) {
	changed := false
	keys.OnEditorChange(gtx, item.targetEnvEditor, func() {
		item.TargetEnvVariable = item.targetEnvEditor.Text()
		changed = true
	})

	keys.OnEditorChange(gtx, &item.onStatusCodeEditor.Editor, func() {
		item.OnStatusCode = item.onStatusCodeEditor.Value()
		changed = true
	})

	keys.OnEditorChange(gtx, item.sourceKeyEditor, func() {
		item.SourceKey = item.sourceKeyEditor.Text()
		changed = true
	})

	keys.OnEditorChange(gtx, item.jsonPathCodeEditor, func() {
		item.JsonPath = item.jsonPathCodeEditor.Text()
		changed = true
	})

	if item.enableBool.Update(gtx) {
		item.Enable = item.enableBool.Value
		changed = true
	}

	if changed {
		f.triggerChanged()

		if f.responseDetail == nil {
			return
		}

		if f.previewTitle != item.TargetEnvVariable {
			f.previewTitle = item.TargetEnvVariable
			f.previewValue = ""
		}

		// its either the http or grpc response available
		if f.requestType == domain.RequestTypeHTTP {
			f.handleHttpPreview(item)
		} else if f.requestType == domain.RequestTypeGRPC {
			f.handleGrpcPreview(item)
		}
	}
}

func (f *Variables) handleGrpcPreview(item *Variable) {
	if f.responseDetail.GRPC == nil {
		return
	}

	if f.responseDetail.GRPC.StatusCode != item.OnStatusCode {
		return
	}

	var (
		pre interface{}
		err error
	)
	switch item.From {
	case domain.VariableFromBody:
		pre, err = jsonpath.Get(f.responseDetail.GRPC.Response, item.JsonPath)
		if err != nil {
			return
		}
	case domain.VariableFromMetaData:
		pre = domain.FindKeyValue(f.responseDetail.GRPC.ResponseMetadata, item.SourceKey)
	case domain.VariableFromTrailers:
		pre = domain.FindKeyValue(f.responseDetail.GRPC.Trailers, item.SourceKey)
	}

	if result, ok := pre.(string); ok {
		f.previewValue = result
	} else {
		f.previewTitle = fmt.Sprintf("%v", pre)
	}

	f.previewTitle = item.TargetEnvVariable
}

func (f *Variables) handleHttpPreview(item *Variable) {
	if f.responseDetail.HTTP == nil {
		return
	}

	if f.responseDetail.HTTP.StatusCode != item.OnStatusCode {
		return
	}

	var (
		pre interface{}
		err error
	)
	switch item.From {
	case domain.VariableFromBody:
		pre, err = jsonpath.Get(f.responseDetail.HTTP.Response, item.JsonPath)
		if err != nil {
			return
		}
	case domain.VariableFromHeader:
		pre = domain.FindKeyValue(f.responseDetail.HTTP.ResponseHeaders, item.SourceKey)
	case domain.VariableFromCookies:
		pre = domain.FindKeyValue(f.responseDetail.HTTP.Cookies, item.SourceKey)
	}

	if result, ok := pre.(string); ok {
		f.previewValue = result
	} else {
		f.previewTitle = fmt.Sprintf("%v", pre)
	}

	f.previewTitle = item.TargetEnvVariable
}
