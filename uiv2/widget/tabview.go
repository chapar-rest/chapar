package widget

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// TabView is a closable tab container built on [core.Tabs] with FunctionalTabs styling.
type TabView struct {
	*core.Tabs
	index    map[string]*core.Frame
	keys     []string
	onChange func(empty bool)
}

// NewTabView creates a TabView as a child of parent.
func NewTabView(parent tree.Node) *TabView {
	tv := &TabView{
		Tabs:  core.NewTabs(parent),
		index: make(map[string]*core.Frame),
	}
	tv.SetName("tab-view")
	tv.Type = core.FunctionalTabs
	tv.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Padding.Zero()
		s.Margin.Zero()
	})
	tv.CloseTabFunc = func(idx int) {
		if idx >= 0 && idx < len(tv.keys) {
			key := tv.keys[idx]
			delete(tv.index, key)
			tv.keys = append(tv.keys[:idx], tv.keys[idx+1:]...)
		}
		tv.notifyChange()
	}
	tree.AddChildInit(tv, "frame", func(f *core.Frame) {
		f.Styler(func(s *styles.Style) {
			s.Overflow.X = styles.OverflowHidden
			s.Min.X.Zero()
		})
	})
	tv.configureTabBar()
	return tv
}

// OnChange installs a callback invoked when tabs are opened or closed. The
// callback receives whether the tab view is now empty.
func (tv *TabView) OnChange(fn func(empty bool)) {
	tv.onChange = fn
}

func (tv *TabView) notifyChange() {
	if tv.onChange != nil {
		tv.onChange(tv.IsEmpty())
	}
}

// IsEmpty reports whether any tabs are open.
func (tv *TabView) IsEmpty() bool {
	return len(tv.keys) == 0
}

// configureTabBar styles the tab strip and wires per-tab styling for new buttons.
func (tv *TabView) configureTabBar() {
	tv.UpdateWidget()
	bar := tv.ChildByName("tabs")
	if bar == nil {
		return
	}
	barFrame := core.AsFrame(bar)
	barFrame.Styler(func(s *styles.Style) {
		s.Gap.Zero()
		s.Padding.Zero()
		s.Justify.Content = styles.Start
		s.Align.Items = styles.End
	})
	bar.AsTree().SetOnChildAdded(func(n tree.Node) {
		if tab, ok := n.(*core.Tab); ok {
			styleFunctionalTab(tab)
			tv.attachTabMenu(tab)
		}
	})
	for i := range barFrame.NumChildren() {
		if tab, ok := barFrame.Child(i).(*core.Tab); ok {
			styleFunctionalTab(tab)
			tv.attachTabMenu(tab)
		}
	}
}

func (tv *TabView) attachTabMenu(tab *core.Tab) {
	tab.AddContextMenu(func(m *core.Scene) {
		name := tab.Name
		core.NewButton(m).SetText("Close").SetIcon(icons.Close).
			OnClick(func(e events.Event) {
				tv.closeTabByName(name)
			})
		core.NewButton(m).SetText("Close others").
			OnClick(func(e events.Event) {
				tv.closeOthers(name)
			})
		core.NewButton(m).SetText("Close all").
			OnClick(func(e events.Event) {
				tv.closeAll()
			})
	})
}

func (tv *TabView) closeTabByName(name string) {
	for i, k := range tv.keys {
		if k == name {
			tv.DeleteTabIndex(i)
			return
		}
	}
}

func (tv *TabView) closeOthers(keep string) {
	for i := len(tv.keys) - 1; i >= 0; i-- {
		if tv.keys[i] != keep {
			tv.DeleteTabIndex(i)
		}
	}
}

func (tv *TabView) closeAll() {
	for len(tv.keys) > 0 {
		tv.DeleteTabIndex(0)
	}
}

// styleFunctionalTab applies Chapar tab-bar styling on top of core defaults.
func styleFunctionalTab(tab *core.Tab) {
	tab.FinalStyler(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmallTop
		s.Padding.Set(units.Dp(4), units.Dp(12), units.Dp(4), units.Dp(12))
		s.Gap.Set(units.Dp(4))

		if s.Is(states.Selected) {
			s.Background = colors.Scheme.SurfaceContainerHigh
			s.Color = colors.Scheme.OnSurface
			s.Border.Style.Bottom = styles.BorderSolid
			s.Border.Width.Bottom = units.Dp(2)
			s.Border.Color.Bottom = colors.Scheme.Primary.Base
		} else {
			s.Background = colors.Scheme.SurfaceContainer
			s.Color = colors.Scheme.OnSurfaceVariant
			s.Border.Style.Bottom = styles.BorderNone
			s.Border.Width.Bottom = units.Zero()
		}
	})
}

func styleTabContentFrame(content *core.Frame) {
	content.Styler(func(s *styles.Style) {
		s.Overflow.X = styles.OverflowHidden
		s.Min.X.Zero()
		s.Grow.Set(1, 1)
	})
}

// Open returns (and focuses) the tab for key. If the tab does not yet exist, it
// is created with label as the visible title and build is called once with the
// tab's content frame. Subsequent Open calls with the same key only focus the tab.
func (tv *TabView) Open(key, label string, build func(content *core.Frame)) *core.Frame {
	if frame, ok := tv.index[key]; ok {
		tv.SelectTabByName(key)
		tv.Update()
		return frame
	}

	content, tabBtn := tv.NewTab(key)
	tabBtn.SetText(label)
	styleFunctionalTab(tabBtn)
	// Context menu is attached in configureTabBar's OnChildAdded when NewTab runs.
	styleTabContentFrame(content)
	tv.index[key] = content
	tv.keys = append(tv.keys, key)
	if build != nil {
		build(content)
	}
	tv.SelectTabByName(key)
	tv.notifyChange()
	tv.Update()
	return content
}

// Close removes the tab if present.
func (tv *TabView) Close(key string) {
	for i, k := range tv.keys {
		if k == key {
			tv.DeleteTabIndex(i)
			return
		}
	}
}

// Select focuses an existing tab. Returns false if key is unknown.
func (tv *TabView) Select(key string) bool {
	if !tv.Has(key) {
		return false
	}
	tv.SelectTabByName(key)
	return true
}

// Has reports whether a tab with key is open.
func (tv *TabView) Has(key string) bool {
	_, ok := tv.index[key]
	return ok
}

// SetTabLabel updates the visible title of the tab identified by key.
func (tv *TabView) SetTabLabel(key, label string) {
	tv.UpdateWidget()
	bar := tv.ChildByName("tabs")
	if bar == nil {
		return
	}
	barFrame := core.AsFrame(bar)
	for i := range barFrame.NumChildren() {
		tab, ok := barFrame.Child(i).(*core.Tab)
		if !ok || tab.Name != key {
			continue
		}
		tab.SetText(label)
		styleFunctionalTab(tab)
		break
	}
	tv.Update()
}
