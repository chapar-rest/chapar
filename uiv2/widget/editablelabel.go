package widget

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// EditableLabel shows title-style text that becomes an outlined text field on click.
// Commit with Enter or when focus leaves; Escape cancels.
type EditableLabel struct {
	core.Frame

	text      string
	labelType core.TextTypes
	editing   bool
	onCommit  func(string)

	field       *core.TextField
	pinnedWidth float32 // content width captured when entering edit mode
}

// NewEditableLabel creates an editable title widget with the given initial text.
func NewEditableLabel(parent tree.Node, text string) *EditableLabel {
	el := tree.New[EditableLabel](parent)
	el.text = text
	el.Update()
	return el
}

// SetLabelType sets the text style used when not editing.
func (el *EditableLabel) SetLabelType(t core.TextTypes) *EditableLabel {
	el.labelType = t
	el.Update()
	return el
}

// SetText updates the displayed label without entering edit mode.
func (el *EditableLabel) SetText(text string) {
	el.text = text
	if el.editing {
		return
	}
	if el.field != nil {
		el.field.SetText(text)
	}
}

// OnCommit installs a callback invoked when the user finishes editing.
func (el *EditableLabel) OnCommit(fn func(string)) *EditableLabel {
	el.onCommit = fn
	return el
}

func (el *EditableLabel) Init() {
	el.Frame.Init()
	if el.labelType == 0 {
		el.labelType = core.TextTitleMedium
	}

	el.Styler(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Min.X.Zero()
		s.SetAbilities(true, abilities.Activatable, abilities.Clickable, abilities.Hoverable)
	})
	el.FinalStyler(func(s *styles.Style) {
		if el.editing {
			s.Cursor = cursors.Text
			s.Background = nil
			return
		}
		s.Cursor = cursors.Pointer
		if s.Is(states.Hovered) {
			s.Background = colors.Scheme.SurfaceContainerHighest
		} else {
			s.Background = nil
		}
	})

	startEdit := func(e events.Event) {
		if !el.editing {
			el.beginEdit()
			e.SetHandled()
		}
	}
	el.OnClick(startEdit)
	el.OnDoubleClick(startEdit)

	el.Maker(func(p *tree.Plan) {
		tree.AddAt(p, "field", func(w *core.TextField) {
			el.field = w
			w.SetType(core.TextFieldOutlined)
			w.SetText(el.text)

			w.OnFirst(events.Click, func(e events.Event) {
				if !el.editing {
					el.beginEdit()
					e.SetHandled()
				}
			})

			w.OnFinal(events.Change, func(e events.Event) {
				if el.editing {
					el.commitEdit()
				}
			})

			w.OnKeyChord(func(e events.Event) {
				if !el.editing {
					return
				}
				switch e.KeyChord() {
				case "ReturnEnter", "KeypadEnter":
					el.commitEdit()
					e.SetHandled()
				case "Escape":
					el.cancelEdit()
					e.SetHandled()
				}
			})

			w.OnFinal(events.FocusLost, func(e events.Event) {
				if el.editing {
					el.commitEdit()
				}
			})

			w.FinalStyler(func(s *styles.Style) {
				s.Grow.Set(0, 0)
				s.SetTextWrap(false)
				s.Padding.Set(units.Dp(2), units.Dp(4))
				applyTextTypeStyle(s, el.labelType)
				if el.editing && el.pinnedWidth > 0 {
					s.Min.X.Dot(el.pinnedWidth)
					s.Max.X.Dot(el.pinnedWidth)
					s.SetAbilities(true, abilities.Focusable)
					s.Background = nil
					return
				}
				s.Min.X.Zero()
				s.Max.X.Ch(0)
				s.SetAbilities(false, abilities.Focusable)
				s.Background = nil
				s.Border.Width.Zero()
				s.Border.Color.Zero()
				s.Border.Style.Set(styles.BorderNone)
			})

			w.Updater(func() {
				w.SetText(el.text)
				w.SetReadOnly(!el.editing)
				w.SetType(core.TextFieldOutlined)
			})
		})
	})
}

func (el *EditableLabel) labelWidth() float32 {
	if el.field == nil {
		return 0
	}
	w := el.field.Geom.Size.Actual.Content.X
	if w <= 0 {
		w = el.field.Geom.Size.Actual.Total.X
	}
	return w
}

func (el *EditableLabel) beginEdit() {
	if el.editing || el.field == nil {
		return
	}
	el.pinnedWidth = el.labelWidth()
	el.editing = true
	el.field.SetReadOnly(false)
	el.field.SetText(el.text)
	el.Restyle()
	el.field.Restyle()
	el.Update()
	el.field.Defer(func() {
		if el.field != nil && el.editing {
			el.field.SetFocus()
		}
	})
}

func (el *EditableLabel) commitEdit() {
	if !el.editing || el.field == nil {
		return
	}
	next := el.field.Text()
	if next == "" {
		next = el.text
	}
	el.text = next
	el.editing = false
	el.pinnedWidth = 0
	el.field.SetReadOnly(true)
	el.field.SetText(next)
	el.clearHoverState()
	el.Restyle()
	el.field.Restyle()
	el.Update()
	if el.onCommit != nil {
		el.onCommit(next)
	}
}

func (el *EditableLabel) cancelEdit() {
	if !el.editing {
		return
	}
	el.editing = false
	el.pinnedWidth = 0
	if el.field != nil {
		el.field.SetText(el.text)
		el.field.SetReadOnly(true)
	}
	el.clearHoverState()
	el.Restyle()
	if el.field != nil {
		el.field.Restyle()
	}
	el.Update()
}

// clearHoverState drops active/hover styling left over from the click that opened edit mode.
func (el *EditableLabel) clearHoverState() {
	el.SetState(false, states.Active|states.Hovered|states.Focused)
	if el.field != nil {
		el.field.SetState(false, states.Active|states.Hovered|states.Focused)
	}
}

func applyTextTypeStyle(s *styles.Style, t core.TextTypes) {
	switch t {
	case core.TextTitleMedium:
		s.Text.LineHeight = 24.0 / 16
		s.Font.Size.Dp(16)
		s.Font.Weight = rich.Bold
	case core.TextTitleSmall:
		s.Text.LineHeight = 20.0 / 14
		s.Font.Size.Dp(14)
		s.Font.Weight = rich.Medium
	case core.TextBodyMedium:
		s.Text.LineHeight = 20.0 / 14
		s.Font.Size.Dp(14)
		s.Font.Weight = rich.Normal
	case core.TextBodyLarge:
		s.Text.LineHeight = 24.0 / 16
		s.Font.Weight = rich.Normal
	default:
		s.Text.LineHeight = 24.0 / 16
		s.Font.Size.Dp(16)
		s.Font.Weight = rich.Bold
	}
}
