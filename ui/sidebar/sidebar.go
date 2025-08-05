package sidebar

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

var (
	hoverOverlayAlpha    uint8 = 48
	selectedOverlayAlpha uint8 = 96
)

type Sidebar struct {
	AlphaPalette

	selectedItem    int
	selectedChanged bool // selected item changed during the last frame
	items           []renderItem

	itemList layout.List
}

func New() *Sidebar {
	m := &Sidebar{
		AlphaPalette: AlphaPalette{
			Hover:    hoverOverlayAlpha,
			Selected: selectedOverlayAlpha,
		},
	}
	return m
}

// AddNavItem inserts a navigation target into the drawer. This should be
// invoked only from the layout thread to avoid nasty race conditions.
func (s *Sidebar) AddNavItem(item Item) {
	s.items = append(s.items, renderItem{
		Item:         item,
		AlphaPalette: &s.AlphaPalette,
	})
	if len(s.items) == 1 {
		s.items[0].selected = true
	}
}

func (s *Sidebar) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	sheet := component.NewSheet()
	return sheet.Layout(gtx, th.Material(), &component.VisibilityAnimation{}, func(gtx C) D {
		return s.LayoutContents(gtx, th)
	})
}

func (s *Sidebar) LayoutContents(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	s.selectedChanged = false
	gtx.Constraints.Min.Y = 0
	s.itemList.Axis = layout.Vertical
	return s.itemList.Layout(gtx, len(s.items), func(gtx C, index int) D {
		gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(70))
		gtx.Constraints.Min = gtx.Constraints.Max
		if s.items[index].Clicked(gtx) {
			s.changeSelected(index)
		}
		dimensions := s.items[index].Layout(gtx, th)
		return dimensions
	})
}

func (s *Sidebar) changeSelected(newIndex int) {
	if newIndex == s.selectedItem && s.items[s.selectedItem].selected {
		return
	}
	s.items[s.selectedItem].selected = false
	s.selectedItem = newIndex
	s.items[s.selectedItem].selected = true
	s.selectedChanged = true
}

func (s *Sidebar) SetSelected(tag interface{}) {
	for i, item := range s.items {
		if item.Tag == tag {
			s.changeSelected(i)
			break
		}
	}
}

func (s *Sidebar) Current() interface{} {
	return s.items[s.selectedItem].Tag
}

func (s *Sidebar) Changed() bool {
	return s.selectedChanged
}
