package component

import (
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Breadcrumb struct {
	ContainerType  string
	CollectionName string
	Title          *widgets.EditableLabel

	onTitleChanged func(title string)
}

func NewBreadcrumb(containerType, collectionName, title string) *Breadcrumb {
	return &Breadcrumb{
		ContainerType:  containerType,
		CollectionName: collectionName,
		Title:          widgets.NewEditableLabel(title),
	}
}

func (b *Breadcrumb) SetOnTitleChanged(f func(title string)) {
	b.onTitleChanged = f
	b.Title.SetOnChanged(f)
}

func (b *Breadcrumb) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	items := make([]layout.FlexChild, 0)
	if b.ContainerType != "" {
		items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Label(theme, theme.TextSize, strings.ToUpper(b.ContainerType))
			l.Font.Weight = font.Bold
			l.Color = widgets.LightGreen // TODO: use theme color
			return l.Layout(gtx)
		}))
	}

	if b.CollectionName != "" {
		items = append(items,
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme, theme.TextSize, b.CollectionName+" /").Layout(gtx)
			}))
	}

	items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return b.Title.Layout(gtx, theme)
	}))

	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEnd, Alignment: layout.Middle}.Layout(gtx, items...)
}
