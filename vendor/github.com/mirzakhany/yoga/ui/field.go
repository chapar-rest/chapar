package ui

import "github.com/mirzakhany/yoga/layout"

// TextField is a single-line text input. value is controlled by the app;
// id keys caret/focus/scroll across frames.
func TextField(id, value string) *Node {
	return &Node{kind: kindTextField, id: id, text: value}
}

func (n *Node) layoutTextField(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "textfield")
	}
	placeholder, password := n.placeholder, n.password
	iconStart, iconEnd := n.iconStart, n.iconEnd
	tf := c.Widget(id, func() any {
		return NewTextInput(TextFieldConfig{
			Placeholder: placeholder,
			Password:    password,
			IconStart:   iconStart,
			IconEnd:     iconEnd,
		})
	}).(*TextInput)
	if tf.cfg.Placeholder == "" && placeholder != "" {
		tf.cfg.Placeholder = placeholder
	}
	if !iconStart.Empty() {
		tf.cfg.IconStart = iconStart
	}
	if !iconEnd.Empty() {
		tf.cfg.IconEnd = iconEnd
	}
	tf.cfg.Password = password
	n.bindTextInput(c, tf)
	// Re-apply height each frame so cached widgets follow theme metrics and
	// an explicit .Height(...) on the node. EditableLabel passes Height via
	// TextFieldConfig and paints into its own frame, so it is unaffected.
	h := c.controlHeight()
	if n.spec.hasH {
		h = n.spec.height
	}
	tf.cfg.Height = h
	tf.host.Style.Height = h
	tf.host.Style.MinHeight = h
	return n.finishTextInput(c, tf)
}

// bindTextInput copies the node's value, callbacks and style to the field.
func (n *Node) bindTextInput(c *Ctx, tf *TextInput) {
	if tf.Value != n.text {
		tf.Value = n.text
		if tf.caret > len(tf.Value) {
			tf.caret = len(tf.Value)
		}
		if tf.selAnchor > len(tf.Value) {
			tf.selAnchor = -1
		}
		// The app changed the value itself, so the history no longer
		// matches it. Values the field reported via OnChange come back
		// equal and keep it.
		tf.resetHistory()
	}
	tf.joined = c.env.joined
	tf.OnChange = n.onChange
	tf.OnSubmit = n.onSubmit
	tf.Highlight = n.highlight
	tf.Suggest = n.suggest
	tf.HoverInfo = n.hoverInfo
	tf.disabled = n.disabled
	tf.visualSpec = c.styles().TextField.merge(n.spec)
	if n.disabled && tf.focused {
		tf.Blur()
	}
}

// finishTextInput lays the field out and applies the node's layout spec.
func (n *Node) finishTextInput(c *Ctx, tf *TextInput) *layout.Element {
	el := tf.Layout(c)
	tf.RegisterSuggest(c)
	if c.Focus() != nil {
		switch {
		case n.focusNow && !n.disabled:
			c.Focus().Focus(tf)
		case n.defaultFocus:
			c.Focus().EnsureFocus(tf)
		}
	}
	if !n.spec.hasW {
		el.Style.MinWidth = tf.minWidth()
	}
	el.Style = applyLayoutSpec(el.Style, n.spec)
	return el
}
