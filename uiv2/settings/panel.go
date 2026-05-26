package settings

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

type itemKind int

const (
	itemHeader itemKind = iota
	itemBool
	itemText
	itemInt
	itemChooser
)

type panelItem struct {
	kind        itemKind
	title       string
	description string
	visible     func(*Data) bool
	boolPtr     func(*Data) *bool
	intPtr      func(*Data) *int
	stringPtr   func(*Data) *string
	chooser     []core.ChooserItem
	textMinX    units.Value
}

func headerItem(title string) panelItem {
	return panelItem{kind: itemHeader, title: title}
}

func boolItem(title, description string, ptr func(*Data) *bool) panelItem {
	return panelItem{kind: itemBool, title: title, description: description, boolPtr: ptr}
}

func boolItemWhen(title, description string, ptr func(*Data) *bool, visible func(*Data) bool) panelItem {
	item := boolItem(title, description, ptr)
	item.visible = visible
	return item
}

func textItem(title, description string, ptr func(*Data) *string) panelItem {
	return panelItem{
		kind:        itemText,
		title:       title,
		description: description,
		stringPtr:   ptr,
		textMinX:    units.Dp(280),
	}
}

func textItemWhen(title, description string, ptr func(*Data) *string, visible func(*Data) bool) panelItem {
	item := textItem(title, description, ptr)
	item.visible = visible
	return item
}

func intItem(title, description string, ptr func(*Data) *int) panelItem {
	return panelItem{kind: itemInt, title: title, description: description, intPtr: ptr}
}

func chooserItem(title, description string, ptr func(*Data) *string, items []core.ChooserItem) panelItem {
	return panelItem{
		kind:        itemChooser,
		title:       title,
		description: description,
		stringPtr:   ptr,
		chooser:     items,
	}
}

func sectionItems(id string) []panelItem {
	switch id {
	case "general":
		return generalPanelItems()
	case "scripting":
		return scriptingPanelItems()
	case "editor":
		return editorPanelItems()
	case "data":
		return dataPanelItems()
	case "appearance":
		return appearancePanelItems()
	default:
		return appearancePanelItems()
	}
}

func buildSettingPanel(parent tree.Node, draft *Data, sectionID string, onChange func()) {
	list := core.NewFrame(parent)
	list.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Gap.Set(units.Dp(4))
	})

	for _, item := range sectionItems(sectionID) {
		item := item
		if item.visible != nil && !item.visible(draft) {
			continue
		}
		if item.kind == itemHeader {
			hdr := core.NewText(list)
			hdr.SetText(item.title)
			hdr.SetType(core.TextTitleSmall)
			hdr.Styler(func(s *styles.Style) {
				s.Padding.Set(units.Dp(12), units.Dp(4), units.Dp(4), units.Dp(4))
			})
			continue
		}
		addSettingRow(list, draft, item, onChange)
	}
}

func addSettingRow(parent tree.Node, draft *Data, item panelItem, onChange func()) {
	row := core.NewFrame(parent)
	row.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Align.Items = styles.Center
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(24))
		s.Padding.Set(units.Dp(10), units.Dp(4))
	})

	left := core.NewFrame(row)
	left.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(4))
	})

	title := core.NewText(left)
	title.SetText(item.title)
	title.SetType(core.TextBodyLarge)
	title.Styler(func(s *styles.Style) {
		s.SetTextWrap(true)
		s.Grow.Set(1, 0)
	})

	desc := core.NewText(left)
	desc.SetText(item.description)
	desc.SetType(core.TextBodySmall)
	desc.Styler(func(s *styles.Style) {
		s.Color = colors.Scheme.OnSurfaceVariant
		s.SetTextWrap(true)
		s.Grow.Set(1, 0)
	})

	right := core.NewFrame(row)
	right.Styler(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Align.Self = styles.Center
	})

	changed := func(e events.Event) {
		onChange()
	}

	switch item.kind {
	case itemBool:
		sw := core.NewSwitch(right).SetType(core.SwitchSwitch)
		core.Bind(item.boolPtr(draft), sw)
		sw.OnChange(changed)
	case itemText:
		tf := core.NewTextField(right)
		tf.Styler(func(s *styles.Style) {
			s.Min.X = item.textMinX
		})
		core.Bind(item.stringPtr(draft), tf)
		tf.OnChange(changed)
	case itemInt:
		sp := core.NewSpinner(right).SetStep(1).SetFormat("%d")
		sp.Styler(func(s *styles.Style) {
			s.Min.X.Dp(120)
		})
		core.Bind(item.intPtr(draft), sp)
		sp.OnChange(changed)
	case itemChooser:
		ch := core.NewChooser(right).SetItems(item.chooser...)
		ch.Styler(func(s *styles.Style) {
			s.Min.X.Dp(180)
		})
		core.Bind(item.stringPtr(draft), ch)
		ch.OnChange(changed)
	}
}
