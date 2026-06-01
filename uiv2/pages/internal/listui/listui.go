package listui

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
)

// SectionHeader renders a small title at the top of a list panel section.
func SectionHeader(panel *core.Frame, title string) {
	core.NewText(panel).
		SetType(core.TextTitleSmall).
		SetText(title).
		Styler(func(s *styles.Style) {
			s.Padding.Set(units.Dp(4), units.Dp(2))
			s.SetTextWrap(false)
		})
}

// ItemOpts configures a list panel row.
type ItemOpts struct {
	OnClick func()
	OnMenu  func(m *core.Scene)
}

// Item renders a clickable row in a list panel.
func Item(panel *core.Frame, label, id string, onClick func()) {
	ItemWith(panel, label, id, ItemOpts{OnClick: onClick})
}

// ItemWith renders a clickable row with an optional context menu.
func ItemWith(panel *core.Frame, label, _ string, opts ItemOpts) {
	row := core.NewFrame(panel)
	row.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Clickable, abilities.Hoverable, abilities.Focusable)
		s.Padding.Set(units.Dp(8), units.Dp(12))
		s.Border.Radius = styles.BorderRadiusSmall
		s.Cursor = cursors.Pointer
		if s.Is(states.Hovered) {
			s.Background = colors.Scheme.SurfaceContainerHighest
		}
	})
	row.OnClick(func(e events.Event) {
		if opts.OnClick != nil {
			opts.OnClick()
		}
	})
	if opts.OnMenu != nil {
		row.AddContextMenu(opts.OnMenu)
	}
	lbl := core.NewText(row).
		SetType(core.TextBodyMedium).
		SetText(label)
	lbl.Styler(func(s *styles.Style) {
		s.SetNonSelectable()
		s.SetTextWrap(false)
		s.Grow.Set(1, 0)
		s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable,
			abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
	})
	lbl.OnClick(func(e events.Event) {
		if opts.OnClick != nil {
			opts.OnClick()
		}
	})
}
