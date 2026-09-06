package ui

import (
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

type editableLabelState struct {
	editing     bool
	hovered     bool
	focused     bool
	draft       string
	el          *layout.Element
	field       *TextInput
	onSave      func(string)
	disabled    bool
	placeholder string
}

func (st *editableLabelState) Focus() {
	st.focused = true
	if st.editing {
		st.field.Focus()
	}
}

func (st *editableLabelState) Blur() {
	if st.editing {
		st.cancelEdit()
	}
	st.focused = false
}

func (st *editableLabelState) Focused() bool { return st.focused }

func (st *editableLabelState) CapturesTab() bool { return false }

func (st *editableLabelState) FocusOnClick() bool { return true }

func (st *editableLabelState) FocusEl() *layout.Element { return st.el }

func (st *editableLabelState) HandleText(runes []rune) {
	if st.editing {
		st.field.HandleText(runes)
		return
	}
	if !st.focused || st.disabled {
		return
	}
	for _, r := range runes {
		if r == ' ' || r == '\n' {
			return
		}
	}
}

func (st *editableLabelState) HandleKeys(keys []input.KeyEvent) {
	if st.editing {
		for _, ev := range keys {
			if ev.Key == input.KeyEscape {
				st.cancelEdit()
				return
			}
			if ev.Key == input.KeyEnter && !ev.Mods.Primary() {
				st.saveEdit()
				return
			}
		}
		st.field.HandleKeys(keys)
		return
	}
	if !st.focused || st.disabled {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEnter || ev.Key == input.KeySpace {
			if ev.Key == input.KeySpace && ev.Mods != 0 {
				continue
			}
			st.startEdit(st.draft)
			return
		}
	}
}

func (st *editableLabelState) startEdit(value string) {
	st.editing = true
	st.draft = value
	st.field.OnChange = func(s string) { st.draft = s }
	st.field.setValue(value)
	st.field.selAnchor = 0
	st.field.caret = len(value)
	st.field.Focus()
	st.focused = true
}

func (st *editableLabelState) cancelEdit() {
	st.editing = false
	st.draft = ""
	st.field.Blur()
	st.field.selAnchor = -1
}

func (st *editableLabelState) saveEdit() {
	if !st.editing {
		return
	}
	val := st.field.Value
	st.editing = false
	st.field.Blur()
	st.field.selAnchor = -1
	if st.onSave != nil {
		st.onSave(val)
	}
}

var _ Focusable = (*editableLabelState)(nil)

func editableLabelHeight(th *theme.Theme) float32 {
	style := th.Typography.Body
	return style.LineHeight + th.Spacing.S*2
}

func editableLabelPadX(th *theme.Theme) float32 {
	return th.Spacing.MNudge
}

func measureEditableLabel(c *Ctx, display string, size float32) (tw, lh float32) {
	if eng := c.Text(); eng != nil {
		return eng.MeasureAt(display, size)
	}
	lh = size
	tw = size * 0.5 * float32(len(display))
	return tw, lh
}

// EditableLabel is a click-to-edit label. Hover shows an I-beam cursor; Enter
// commits via OnSave; Escape or blur cancels without saving.
func EditableLabel(id, value string) *Node {
	return &Node{kind: kindEditableLabel, id: id, text: value}
}

func (n *Node) layoutEditableLabel(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "editablelabel")
	}
	th := c.Theme()
	h := editableLabelHeight(th)
	padX := editableLabelPadX(th)
	style := th.Typography.Body

	st := c.Widget(id, func() any {
		field := NewTextInput(TextFieldConfig{Height: h})
		return &editableLabelState{field: field}
	}).(*editableLabelState)

	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	st.onSave = n.onSubmit
	st.disabled = n.disabled
	st.placeholder = n.placeholder
	st.field.cfg.Height = h

	value := n.text
	if !st.editing {
		st.draft = value
	}

	display := value
	if display == "" {
		display = n.placeholder
	}
	tw, _ := measureEditableLabel(c, display, style.Size)
	minW := tw + 2*padX

	box := layout.Box().H(h).Min(minW, h).FlexShrink(0)
	el := layout.New(applyLayoutSpec(box, n.spec))
	st.el = el

	labelValue := value
	placeholder := n.placeholder
	disabled := n.disabled
	radius := th.Radius.Medium

	if st.editing {
		st.field.Update(c.Mouse())
		if st.field.focused {
			since := time.Since(st.field.blinkStart) % (2 * textFieldBlink)
			wait := textFieldBlink - (since % textFieldBlink)
			c.Animate(wait)
		}
		el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
			st.field.host.Frame = el.Frame
			st.field.paint(dl, eng)
		}
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			if disabled {
				return
			}
			st.field.host.Frame = e.Frame
			st.field.onMouse(st.field.host, m)
			inside := e.Frame.Contains(m.X, m.Y)
			st.hovered = inside
			if inside {
				m.SetCursor(CursorText)
			}
		}
	} else {
		el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
			f := el.Frame
			if st.hovered && !disabled {
				dl.AddRoundedRect(f, radius, th.ListHover)
			}
			show := labelValue
			col := th.Foreground
			if show == "" {
				show = placeholder
				col = th.ForegroundMuted
			}
			if show == "" {
				return
			}
			_, lh := eng.MeasureAt(show, style.Size)
			tx := f.X + padX
			ty := f.Y + (f.H-lh)/2
			eng.DrawStringTopAt(dl, show, tx, ty, col, style.Size)
			if st.focused {
				paintFocusRing(dl, f, th.Surface, th)
			}
		}
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			if disabled {
				return
			}
			inside := e.Frame.Contains(m.X, m.Y)
			st.hovered = inside
			if inside {
				m.SetCursor(CursorText)
			}
			if inside && m.Released {
				st.startEdit(labelValue)
				m.Consumed = true
			}
			if inside && m.Pressed {
				m.Consumed = true
			}
		}
	}

	if n.defaultFocus && c.Focus() != nil {
		c.Focus().EnsureFocus(st)
	}

	return el
}
