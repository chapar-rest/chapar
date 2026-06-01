package widget

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// EditableLabel shows title-style text that becomes an outlined text field on click.
// Commit with Enter or when focus leaves; Escape cancels.
type EditableLabel struct {
	core.TextField

	text        string
	labelType   core.TextTypes
	editing     bool
	pinnedWidth float32 // content width captured when entering edit mode
	onCommit    func(string)
}

// NewEditableLabel creates an editable title widget with the given initial text.
func NewEditableLabel(parent tree.Node, text string) *EditableLabel {
	el := tree.New[EditableLabel](parent)
	el.text = text
	el.TextField.SetText(text)
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
	el.TextField.SetText(text)
}

// OnCommit installs a callback invoked when the user finishes editing.
func (el *EditableLabel) OnCommit(fn func(string)) *EditableLabel {
	el.onCommit = fn
	return el
}

func (el *EditableLabel) Init() {
	el.TextField.Init()
	if el.labelType == 0 {
		el.labelType = core.TextTitleMedium
	}
	el.SetType(core.TextFieldOutlined)
	el.SetReadOnly(true)

	el.Styler(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.SetTextWrap(false)
		s.Padding.Set(units.Dp(2), units.Dp(4))
		applyTextTypeStyle(s, el.labelType)
		if el.editing && el.pinnedWidth > 0 {
			s.Min.X.Dot(el.pinnedWidth)
			s.Max.X.Dot(el.pinnedWidth)
		} else {
			s.Min.X.Zero()
			s.Max.X.Ch(0)
		}
		if el.editing {
			s.Cursor = cursors.Text
		} else {
			s.Cursor = cursors.Pointer
		}
	})

	el.Updater(func() {
		if !el.editing {
			el.TextField.SetText(el.text)
			el.SetReadOnly(true)
		}
	})

	el.OnClick(func(e events.Event) {
		if !el.editing {
			el.beginEdit()
			e.SetHandled()
		}
	})
	el.OnDoubleClick(func(e events.Event) {
		if !el.editing {
			el.beginEdit()
			e.SetHandled()
		}
	})

	el.OnKeyChord(func(e events.Event) {
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

	el.OnFinal(events.FocusLost, func(e events.Event) {
		if el.editing {
			el.commitEdit()
		}
	})
}

func (el *EditableLabel) labelWidth() float32 {
	w := el.Geom.Size.Actual.Content.X
	if w <= 0 {
		w = el.Geom.Size.Actual.Total.X
	}
	return w
}

func (el *EditableLabel) beginEdit() {
	if el.editing {
		return
	}
	el.pinnedWidth = el.labelWidth()
	el.editing = true
	el.SetReadOnly(false)
	el.TextField.SetText(el.text)
	el.Restyle()
	el.Update()
	el.Defer(func() {
		if el.editing {
			el.SetFocus()
		}
	})
}

func (el *EditableLabel) commitEdit() {
	if !el.editing {
		return
	}
	next := el.TextField.Text()
	if next == "" {
		next = el.text
	}
	el.text = next
	el.editing = false
	el.pinnedWidth = 0
	el.SetReadOnly(true)
	el.TextField.SetText(next)
	el.clearHoverState()
	el.Restyle()
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
	el.TextField.SetText(el.text)
	el.SetReadOnly(true)
	el.clearHoverState()
	el.Restyle()
	el.Update()
}

// clearHoverState drops active/hover styling left over from the click that opened edit mode.
func (el *EditableLabel) clearHoverState() {
	el.SetState(false, states.Active|states.Hovered|states.Focused)
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
