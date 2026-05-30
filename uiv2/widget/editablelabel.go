package widget

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// EditableLabel shows text that becomes a text field when clicked.
type EditableLabel struct {
	core.Frame

	text      string
	labelType core.TextTypes
	editing   bool
	onCommit  func(string)
}

// NewEditableLabel creates an editable title widget with the given initial text.
func NewEditableLabel(parent tree.Node, text string) *EditableLabel {
	el := tree.New[EditableLabel](parent)
	el.text = text
	el.labelType = core.TextTitleMedium
	el.showLabel()
	return el
}

// SetLabelType sets the text style used when not editing.
func (el *EditableLabel) SetLabelType(t core.TextTypes) *EditableLabel {
	el.labelType = t
	if !el.editing {
		el.showLabel()
	}
	return el
}

// SetText updates the displayed label without entering edit mode.
func (el *EditableLabel) SetText(text string) {
	el.text = text
	if !el.editing {
		el.showLabel()
	}
}

// OnCommit installs a callback invoked when the user finishes editing.
func (el *EditableLabel) OnCommit(fn func(string)) *EditableLabel {
	el.onCommit = fn
	return el
}

func (el *EditableLabel) showLabel() {
	el.editing = false
	el.DeleteChildren()
	el.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Clickable, abilities.Hoverable, abilities.Focusable)
		s.Cursor = cursors.Pointer
		s.Grow.Set(1, 0)
		if s.Is(states.Hovered) {
			s.Background = colors.Scheme.SurfaceContainerHighest
		}
	})
	lbl := core.NewText(el).
		SetType(el.labelType).
		SetText(el.text)
	lbl.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	el.OnClick(func(e events.Event) {
		el.showEditor()
	})
	el.Update()
}

func (el *EditableLabel) showEditor() {
	el.editing = true
	el.DeleteChildren()
	el.Styler(func(s *styles.Style) {
		s.Cursor = cursors.Text
		s.Grow.Set(1, 0)
	})

	tf := core.NewTextField(el)
	tf.SetText(el.text)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Ch(16)
		s.Grow.Set(1, 0)
	})

	commit := func() {
		next := tf.Text()
		if next == "" {
			next = el.text
		}
		el.text = next
		if el.onCommit != nil {
			el.onCommit(next)
		}
		el.showLabel()
	}

	tf.OnChange(func(e events.Event) {
		commit()
	})

	el.Update()
	tf.SetFocus()
}
