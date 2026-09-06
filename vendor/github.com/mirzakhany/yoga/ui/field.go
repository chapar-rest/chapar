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
	if tf.Value != n.text {
		tf.Value = n.text
		if tf.caret > len(tf.Value) {
			tf.caret = len(tf.Value)
		}
		if tf.selAnchor > len(tf.Value) {
			tf.selAnchor = -1
		}
	}
	tf.OnChange = n.onChange
	tf.OnSubmit = n.onSubmit
	tf.disabled = n.disabled
	tf.visualSpec = c.styles().TextField.merge(n.spec)
	if n.disabled && tf.focused {
		tf.Blur()
	}
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
	el := tf.Layout(c)
	if n.defaultFocus && c.Focus() != nil {
		c.Focus().EnsureFocus(tf)
	}
	if !n.spec.hasW {
		el.Style.MinWidth = tf.minWidth()
	}
	el.Style = applyLayoutSpec(el.Style, n.spec)
	return el
}
