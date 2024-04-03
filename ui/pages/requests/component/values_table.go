package component

import (
	"fmt"

	"github.com/mirzakhany/chapar/internal/domain"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ValuesTable struct {
	Title  string
	Values []KeyValue

	list *widget.List
}

type KeyValue struct {
	Key   string
	Value string

	keySelectable   widget.Selectable
	valueSelectable widget.Selectable
}

func NewValuesTable(Title string, values []KeyValue) *ValuesTable {
	return &ValuesTable{
		Title:  Title,
		Values: values,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (v *ValuesTable) SetData(values []domain.KeyValue) {
	v.Values = make([]KeyValue, len(values))
	for i, kv := range values {
		v.Values[i] = KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		}
	}
}
func (v *ValuesTable) GetData() []domain.KeyValue {
	values := make([]domain.KeyValue, len(v.Values))
	for i, kv := range v.Values {
		values[i] = domain.KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		}
	}
	return values
}

func (v *ValuesTable) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if len(v.Values) == 0 {
		return Message(gtx, theme, fmt.Sprintf("No %s available", v.Title))
	}

	return material.List(theme, v.list).Layout(gtx, len(v.Values), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(theme, theme.TextSize, v.Values[i].Key+":")
					l.Font.Weight = font.Bold
					l.State = &v.Values[i].keySelectable
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(theme, theme.TextSize, v.Values[i].Value)
					l.State = &v.Values[i].valueSelectable
					return l.Layout(gtx)
				})
			}),
		)
	})
}
