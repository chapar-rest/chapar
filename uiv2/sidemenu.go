package uiv2

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

// sideMenuItemFrame is a [core.Frame] subtype that anchors its tooltip to the
// right side of the item, vertically centered, instead of the default
// top-center anchor (which makes tooltips appear in the top-right corner for
// narrow sidebar items).
type sideMenuItemFrame struct {
	core.Frame
}

// WidgetTooltip overrides the default [core.WidgetBase.WidgetTooltip] anchor.
// It returns a point just to the right of the widget; the tooltip popup
// positioner then places the tooltip body above that anchor, which puts it
// roughly at the vertical middle and right of the item.
func (f *sideMenuItemFrame) WidgetTooltip(pos image.Point) (string, image.Point) {
	bb := f.Geom.TotalBBox
	if f.Scene != nil {
		bb = bb.Add(f.Scene.SceneGeom.Pos)
	}
	anchor := image.Point{
		X: bb.Max.X + 8,
		Y: bb.Max.Y,
	}
	return f.Tooltip, anchor
}

// SideMenuItem describes a single navigation entry in the [SideMenu].
type SideMenuItem struct {
	// Tag is the value reported by [SideMenu.Current] and passed to
	// [SideMenu.OnSelect] handlers. It identifies the item to caller code.
	Tag any
	// Name is the label rendered beneath the icon.
	Name string
	// Icon is the optional icon rendered above the label.
	Icon icons.Icon
}

// SideMenu is a vertical navigation column, modeled after the legacy
// gio-based [ui/sidebar.Sidebar]. Items are stacked top-to-bottom with the
// icon over a small label; exactly one item is selected at a time.
//
// The menu is rendered as a regular child of the parent passed to
// [NewSideMenu]. The parent is responsible for laying it out (typically by
// placing it on the left of a row-oriented body or frame).
type SideMenu struct {
	bar      *core.Frame
	items    []sideMenuEntry
	current  int
	onSelect func(item SideMenuItem)
}

type sideMenuEntry struct {
	item  SideMenuItem
	frame *core.Frame
}

// NewSideMenu creates a SideMenu as a child of parent. The bar is an 80dp
// column with a contrasting background, designed to sit on the left edge of
// a row-oriented container.
func NewSideMenu(parent tree.Node) *SideMenu {
	m := &SideMenu{current: -1}
	m.bar = core.NewFrame(parent)
	m.bar.SetName("side-menu")
	m.bar.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(0, 1)
		s.Min.X.Dp(72)
		s.Max.X.Dp(72)
		s.Padding.Set(units.Dp(12), units.Dp(4), units.Dp(6), units.Dp(4))
		s.Gap.Set(units.Dp(4))
		s.Justify.Content = styles.Start
		s.Align.Items = styles.Center
		s.Background = colors.Scheme.SurfaceContainerLow
		s.Border.Style.Right = styles.BorderSolid
		s.Border.Width.Right = units.Dp(1)
		s.Border.Color.Right = colors.Scheme.OutlineVariant
		s.Overflow.Y = styles.OverflowAuto
	})
	return m
}

// AddItem registers a navigation entry and renders it in the bar.
func (m *SideMenu) AddItem(item SideMenuItem) *SideMenu {
	idx := len(m.items)
	frame := m.buildItem(item, idx)
	m.items = append(m.items, sideMenuEntry{item: item, frame: frame})
	if m.current == -1 {
		m.selectIndex(idx, false)
	}
	return m
}

// OnSelect installs a handler invoked whenever the user picks a new item.
// It is not fired for the initial default selection.
func (m *SideMenu) OnSelect(fn func(item SideMenuItem)) *SideMenu {
	m.onSelect = fn
	return m
}

// Current returns the [SideMenuItem.Tag] of the selected item, or nil if no
// items have been registered.
func (m *SideMenu) Current() any {
	if m.current < 0 || m.current >= len(m.items) {
		return nil
	}
	return m.items[m.current].item.Tag
}

// Select activates the item whose tag matches the given value, firing
// [SideMenu.OnSelect]. It is a no-op if no such item exists.
func (m *SideMenu) Select(tag any) {
	for i, e := range m.items {
		if e.item.Tag == tag {
			m.selectIndex(i, true)
			return
		}
	}
}

func (m *SideMenu) selectIndex(idx int, fire bool) {
	if idx == m.current || idx < 0 || idx >= len(m.items) {
		return
	}
	if m.current >= 0 && m.current < len(m.items) {
		m.items[m.current].frame.SetSelected(false)
	}
	m.current = idx
	m.items[idx].frame.SetSelected(true)
	if fire && m.onSelect != nil {
		m.onSelect(m.items[idx].item)
	}
}

func (m *SideMenu) buildItem(item SideMenuItem, idx int) *core.Frame {
	itemNode := tree.New[sideMenuItemFrame](m.bar)
	frame := &itemNode.Frame
	frame.SetTooltip(item.Name)
	frame.Styler(func(s *styles.Style) {
		s.SetAbilities(true,
			abilities.Activatable,
			abilities.Clickable,
			abilities.Hoverable,
			abilities.Focusable,
			abilities.Selectable,
		)
		s.Direction = styles.Column
		s.CenterAll()
		s.Grow.Set(1, 0)
		s.Padding.Set(units.Dp(6), units.Dp(4))
		s.Gap.Set(units.Dp(2))
		s.Border.Radius = styles.BorderRadiusSmall
		s.Cursor = cursors.Pointer
		if s.Is(states.Hovered) && !s.Is(states.Selected) {
			s.Background = colors.Scheme.SurfaceContainerHighest
		}
	})

	if item.Icon.IsSet() {
		ic := core.NewIcon(frame).SetIcon(item.Icon)
		ic.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Dp(20))
			s.Min.Set(units.Dp(20))
			s.Max.Set(units.Dp(20))
			s.Align.Self = styles.Center
			s.Justify.Self = styles.Center
		})
	}

	lbl := core.NewText(frame).SetText(item.Name).SetType(core.TextLabelSmall)
	lbl.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
		s.Text.Align = text.Center
		s.Font.Size.Dp(11)
		s.Align.Self = styles.Center
		s.Justify.Self = styles.Center
	})

	frame.OnClick(func(e events.Event) {
		m.selectIndex(idx, true)
	})

	return frame
}
