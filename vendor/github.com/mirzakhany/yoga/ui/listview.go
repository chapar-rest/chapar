package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

// ListViewConfig configures a scrollable vertical list.
type ListViewConfig struct {
	// Gap between items; zero uses theme spacing S.
	Gap float32
}

// ListView is a vertically scrollable list with a built-in scrollbar. Add items
// via Add or SetItems; call Update each frame for wheel and thumb interaction.
type ListView struct {
	host   *layout.Element
	scroll *ScrollView
	stack  *layout.Element
	items  []*layout.Element
	gap    float32
}

// NewListView creates an empty scrollable list.
func NewListView(cfg ListViewConfig) *ListView {
	th := theme.Current()
	gap := cfg.Gap
	if gap <= 0 {
		gap = th.Spacing.S
	}
	lv := &ListView{gap: gap}
	lv.stack = layout.New(layout.Box().Direction(layout.Column).Gap(gap))
	lv.scroll = NewScrollView(lv.stack)
	lv.host = lv.scroll.host
	return lv
}

// Add appends one or more items to the list.
func (lv *ListView) Add(items ...*layout.Element) {
	if len(items) == 0 {
		return
	}
	lv.items = append(lv.items, items...)
	lv.sync()
}

// SetItems replaces all list items.
func (lv *ListView) SetItems(items []*layout.Element) {
	lv.items = append([]*layout.Element(nil), items...)
	lv.sync()
}

// Clear removes every item.
func (lv *ListView) Clear() {
	if len(lv.items) == 0 {
		return
	}
	lv.items = lv.items[:0]
	lv.sync()
}

// Remove deletes the item at index i.
func (lv *ListView) Remove(i int) bool {
	if i < 0 || i >= len(lv.items) {
		return false
	}
	lv.items = append(lv.items[:i], lv.items[i+1:]...)
	lv.sync()
	return true
}

// Len reports the number of items.
func (lv *ListView) Len() int { return len(lv.items) }

// Item returns the element at index i, or nil when out of range.
func (lv *ListView) Item(i int) *layout.Element {
	if i < 0 || i >= len(lv.items) {
		return nil
	}
	return lv.items[i]
}

func (lv *ListView) sync() {
	lv.stack.Children = lv.items
	lv.stack.MarkDirty()
}

// Update drives scroll wheel and thumb drag. Call after layout each frame.
func (lv *ListView) Update(m *input.Mouse) {
	lv.scroll.Update(m)
}

func (lv *ListView) Layout(c *Ctx) *layout.Element {
	lv.Update(c.Mouse())
	return lv.host
}
