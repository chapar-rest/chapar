package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// Focusable is a widget that can receive keyboard focus and join tab traversal.
type Focusable interface {
	Focus()
	Blur()
	Focused() bool
	HandleText(runes []rune)
	HandleKeys(keys []input.KeyEvent)
	CapturesTab() bool
	FocusOnClick() bool
	FocusEl() *layout.Element
}

// FocusScope routes Tab traversal and keyboard input among focusables. It is
// rebuilt every frame: widgets call Add during Layout(c), so beginFrame
// clears the per-frame item list while preserving which widget is focused.
type FocusScope struct {
	items        []Focusable
	focused      Focusable // identity preserved across rebuilds
	modal        Focusable // when set, Tab/keys stay inside the modal region
	modalFrom    int       // items[modalFrom:] are descendants of the open modal
	defaultFocus Focusable // first EnsureFocus this frame; applied in finishBuild
}

// NewFocusScope creates an empty scope. The runtime owns one for the lifetime
// of the app and resets its item list each frame.
func NewFocusScope() *FocusScope { return &FocusScope{} }

// beginFrame clears the per-frame registration list. The focused widget is
// kept; it is re-matched against the new item list as components re-Add.
func (f *FocusScope) beginFrame() {
	f.items = f.items[:0]
	f.modal = nil
	f.modalFrom = 0
	f.defaultFocus = nil
}

// BeginModal marks the start of the modal descendant window. Widgets that
// Add after this call (until SetModal) receive Tab and keys while the modal
// is open. Call immediately before laying out dialog content.
func (f *FocusScope) BeginModal() {
	f.modalFrom = len(f.items)
}

// SetModal traps keyboard routing until the next frame so page widgets behind
// a scrim do not receive keys. When no descendants were registered after
// BeginModal (message/input DialogHost), keys go to w. Otherwise Tab cycles
// among those descendants and focus is not stolen from them.
func (f *FocusScope) SetModal(w Focusable) {
	f.modal = w
	if w == nil {
		return
	}
	if len(f.modalItems()) == 0 {
		f.focusTo(w)
		return
	}
	if f.inModal(f.focused) {
		return
	}
	if f.defaultFocus != nil && f.inModal(f.defaultFocus) {
		f.focusTo(f.defaultFocus)
		return
	}
	f.focusTo(f.modalItems()[0])
}

// Add registers a focusable in tab order for this frame. Idempotent per frame
// in practice because each component Adds itself once during its Layout(c).
func (f *FocusScope) Add(items ...Focusable) {
	for _, it := range items {
		if it != nil {
			f.items = append(f.items, it)
		}
	}
}

// Current returns the focused widget, or nil.
func (f *FocusScope) Current() Focusable { return f.focused }

// Focus moves keyboard focus to w (blurring the previous holder).
func (f *FocusScope) Focus(w Focusable) { f.focusTo(w) }

// EnsureFocus records fallback as the default focus target for this frame.
// The choice is applied in finishBuild after every widget has registered, so a
// DefaultFocus control laid out early cannot steal focus from a widget the user
// already clicked (e.g. an editor below a URL field).
func (f *FocusScope) EnsureFocus(fallback Focusable) {
	if fallback != nil && f.defaultFocus == nil {
		f.defaultFocus = fallback
	}
}

// finishBuild resolves focus after the frame's widget list is complete.
// If the current widget re-registered, it is kept. Otherwise the EnsureFocus
// fallback is used (or focus is cleared).
func (f *FocusScope) finishBuild() {
	if f.modal != nil {
		if f.focused != nil && (f.inModal(f.focused) || f.focused == f.modal) {
			return
		}
		if f.defaultFocus != nil && (f.inModal(f.defaultFocus) || f.defaultFocus == f.modal) {
			f.focusTo(f.defaultFocus)
			return
		}
		if items := f.modalItems(); len(items) > 0 {
			f.focusTo(items[0])
			return
		}
		f.focusTo(f.modal)
		return
	}
	if f.focused != nil && f.indexOf(f.focused) >= 0 {
		return
	}
	if f.defaultFocus != nil {
		f.focusTo(f.defaultFocus)
		return
	}
	if f.focused != nil {
		f.focusTo(nil)
	}
}

// focusTo blurs the previous widget and focuses w (which may be nil).
func (f *FocusScope) focusTo(w Focusable) {
	if f.focused == w {
		return
	}
	if f.focused != nil {
		f.focused.Blur()
	}
	f.focused = w
	if w != nil {
		w.Focus()
	}
}

// indexOf returns the position of the focused widget in this frame's item list,
// or -1 if it is no longer present.
func (f *FocusScope) indexOf(w Focusable) int {
	for i, it := range f.items {
		if it == w {
			return i
		}
	}
	return -1
}

func (f *FocusScope) modalItems() []Focusable {
	if f.modal == nil || f.modalFrom >= len(f.items) {
		return nil
	}
	return f.items[f.modalFrom:]
}

func (f *FocusScope) inModal(w Focusable) bool {
	if w == nil {
		return false
	}
	for _, it := range f.modalItems() {
		if it == w {
			return true
		}
	}
	return false
}

func (f *FocusScope) tabItems() []Focusable {
	if f.modal == nil {
		return f.items
	}
	return f.modalItems()
}

// Next advances focus to the next registered widget, wrapping around.
func (f *FocusScope) Next() {
	items := f.tabItems()
	if len(items) == 0 {
		return
	}
	i := -1
	for j, it := range items {
		if it == f.focused {
			i = j
			break
		}
	}
	f.focusTo(items[(i+1+len(items))%len(items)])
}

// Prev moves focus to the previous registered widget, wrapping around.
func (f *FocusScope) Prev() {
	items := f.tabItems()
	if len(items) == 0 {
		return
	}
	i := 0
	for j, it := range items {
		if it == f.focused {
			i = j
			break
		}
	}
	f.focusTo(items[(i-1+len(items))%len(items)])
}

// HandleMouse grants focus on primary click when the pointer is inside a
// FocusOnClick widget's FocusEl. Later items win over earlier ones (topmost).
func (f *FocusScope) HandleMouse(m *input.Mouse) {
	if m == nil || !m.Pressed {
		return
	}
	items := f.tabItems()
	for i := len(items) - 1; i >= 0; i-- {
		it := items[i]
		if !it.FocusOnClick() {
			continue
		}
		if el := it.FocusEl(); el != nil && el.Frame.Contains(m.X, m.Y) {
			f.focusTo(it)
			return
		}
	}
}

// Route delivers keyboard input to the focused widget. Plain Tab moves focus
// unless the current widget CapturesTab(); Ctrl+Tab always moves focus.
// While a modal is open, Escape is always delivered to the modal host; other
// keys stay inside the modal descendants (or the host when there are none).
func (f *FocusScope) Route(kb *input.Keyboard) {
	if kb == nil {
		return
	}
	if f.modal != nil {
		f.routeModal(kb)
		return
	}
	cur := f.focused
	// Don't deliver to a widget that did not re-register this frame (stale focus
	// after a page/tab swap); wait for EnsureFocus or a click to re-establish.
	if cur == nil || f.indexOf(cur) < 0 {
		return
	}
	f.routeTo(cur, kb.Chars, kb.Keys)
}

func (f *FocusScope) routeModal(kb *input.Keyboard) {
	var rest []input.KeyEvent
	for _, ev := range kb.Keys {
		if ev.Key == input.KeyEscape {
			f.modal.HandleKeys([]input.KeyEvent{ev})
			continue
		}
		rest = append(rest, ev)
	}
	items := f.modalItems()
	if len(items) == 0 {
		if len(kb.Chars) > 0 {
			f.modal.HandleText(kb.Chars)
		}
		if len(rest) > 0 {
			f.modal.HandleKeys(rest)
		}
		return
	}
	cur := f.focused
	if !f.inModal(cur) {
		cur = items[0]
		f.focusTo(cur)
	}
	f.routeTo(cur, kb.Chars, rest)
}

func (f *FocusScope) routeTo(cur Focusable, chars []rune, keys []input.KeyEvent) {
	if cur == nil {
		return
	}
	var delivered []input.KeyEvent
	for _, ev := range keys {
		if ev.Key == input.KeyTab {
			if ev.Mods.Has(input.ModCtrl) || !cur.CapturesTab() {
				if ev.Mods.Has(input.ModShift) {
					f.Prev()
				} else {
					f.Next()
				}
				cur = f.focused
				continue
			}
		}
		delivered = append(delivered, ev)
	}
	if cur == nil {
		return
	}
	if len(chars) > 0 {
		cur.HandleText(chars)
	}
	if len(delivered) > 0 {
		cur.HandleKeys(delivered)
	}
}
