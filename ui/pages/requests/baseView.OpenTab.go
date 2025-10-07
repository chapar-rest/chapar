package requests

import (
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/widgets"
)

func (v *BaseView) OpenTab(id, name, tabType string) {
	tab := &widgets.Tab{
		Title:          name,
		Closable:       true,
		CloseClickable: &widget.Clickable{},
		Identifier:     id,
		Meta:           safemap.New[string](),
	}
	tab.Meta.Set(TypeMeta, tabType)

	i := v.tabHeader.AddTab(tab)
	v.tabHeader.SetSelected(i)
}
