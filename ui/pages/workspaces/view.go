package workspaces

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type View struct {
	newButton widget.Clickable
	searchBox *widgets.TextField

	mx            *sync.Mutex
	filterText    string
	filteredItems []*Item

	items []*Item
	list  *widget.List

	onNew    func()
	onDelete func(w *domain.Workspace)
	onUpdate func(w *domain.Workspace)
}

type Item struct {
	deleteButton widget.Clickable

	Name     *widgets.EditableLabel
	readOnly bool

	w *domain.Workspace
}

func NewView() *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	v := &View{
		mx:        &sync.Mutex{},
		searchBox: search,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		filteredItems: make([]*Item, 0),
	}

	v.searchBox.SetOnTextChange(func(text string) {
		if v.items == nil {
			return
		}

		v.Filter(text)
	})

	return v
}

func (v *View) showError(err error) {
	fmt.Println("error", err)
}

func (v *View) SetOnNew(f func()) {
	v.onNew = f
}

func (v *View) SetOnDelete(f func(w *domain.Workspace)) {
	v.onDelete = f
}

func (v *View) SetOnUpdate(f func(w *domain.Workspace)) {
	v.onUpdate = f
}

func (v *View) Filter(text string) {
	v.mx.Lock()
	defer v.mx.Unlock()

	v.filterText = text

	if text == "" {
		v.filteredItems = []*Item{}
		return
	}

	var items []*Item
	for _, item := range v.items {
		if strings.Contains(item.w.MetaData.Name, text) || strings.Contains(item.w.MetaData.ID, text) {
			items = append(items, item)
		}
	}
	v.filteredItems = items
}

func (v *View) SetItems(items []*domain.Workspace) {
	v.items = make([]*Item, 0)
	for _, w := range items {
		v.AddItem(w)
	}

	sort.Slice(v.items, func(i, j int) bool {
		return v.items[i].w.MetaData.Name < v.items[j].w.MetaData.Name
	})
}

func (v *View) RemoveItem(item *domain.Workspace) {
	for i, it := range v.items {
		if it.w.MetaData.ID == item.MetaData.ID {
			v.items = append(v.items[:i], v.items[i+1:]...)
			break
		}
	}
}

func (v *View) AddItem(item *domain.Workspace) {
	readonly := item.MetaData.Name == domain.DefaultWorkspaceName
	nameEditable := widgets.NewEditableLabel(item.MetaData.Name)
	nameEditable.SetReadOnly(readonly)

	nameEditable.SetOnChanged(func(text string) {
		if v.onUpdate != nil {
			item.MetaData.Name = text
			v.onUpdate(item)
		}
	})

	v.items = append(v.items, &Item{w: item, Name: nameEditable, readOnly: readonly})

	sort.Slice(v.items, func(i, j int) bool {
		return v.items[i].w.MetaData.Name < v.items[j].w.MetaData.Name
	})
}

func (v *View) itemLayout(gtx layout.Context, theme *chapartheme.Theme, item *Item, isLast bool) layout.Dimensions {
	content := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(100)
				return item.Name.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if item.readOnly {
					return layout.Dimensions{}
				}

				ib := widgets.IconButton{
					Icon:      widgets.DeleteIcon,
					Size:      unit.Dp(20),
					Color:     theme.TextColor,
					Clickable: &item.deleteButton,
				}

				ib.OnClick = func() {
					if v.onDelete != nil {
						v.onDelete(item.w)
					}
				}

				return ib.Layout(gtx, theme)
			}),
		)
	})

	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return content
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// only if it's not the last item
			if isLast {
				return layout.Dimensions{}
			}
			return widgets.DrawLine(gtx, theme.TableBorderColor, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X))
		}),
	)
}

func (v *View) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	items := v.items
	if v.filterText != "" {
		items = v.filteredItems
	}

	if v.onNew != nil {
		if v.newButton.Clicked(gtx) {
			v.onNew()
		}
	}

	return layout.Inset{Top: unit.Dp(30), Left: unit.Dp(250), Right: unit.Dp(250)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceEnd}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(theme.Material(), unit.Sp(18), "Workspaces")
						lb.Font.Weight = font.Bold
						return lb.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						newBtn := widgets.Button(theme.Material(), &v.newButton, widgets.PlusIcon, widgets.IconPositionStart, "New Workspace")
						newBtn.Color = theme.ButtonTextColor
						newBtn.Background = theme.SendButtonBgColor
						return newBtn.Layout(gtx, theme)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(theme.Material(), theme.TextSize, "Workspaces are a way to organize your work.\nYou can create multiple workspaces to separate your projects.")
						return lb.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = gtx.Dp(200)
						return v.searchBox.Layout(gtx, theme)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(30)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.List(theme.Material(), v.list).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
					return v.itemLayout(gtx, theme, items[i], i == len(items)-1)
				})
			}),
		)
	})
}
