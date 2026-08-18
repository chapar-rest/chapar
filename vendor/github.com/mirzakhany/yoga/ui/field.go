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
	if iconStart != "" {
		tf.cfg.IconStart = iconStart
	}
	if iconEnd != "" {
		tf.cfg.IconEnd = iconEnd
	}
	tf.cfg.Password = password
	if tf.Value != n.text {
		tf.Value = n.text
		if tf.caret > len(tf.Value) {
			tf.caret = len(tf.Value)
		}
	}
	tf.OnChange = n.onChange
	tf.OnSubmit = n.onSubmit
	el := tf.Layout(c)
	if n.defaultFocus && c.Focus() != nil {
		c.Focus().EnsureFocus(tf)
	}
	el.Style = applyLayoutSpec(el.Style, n.spec)
	return el
}
