package component

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

// TruncatingLabelStyle is a type that forces a label to
// fit on one line and adds a truncation indicator symbol
// to the end of the line if the text has been truncated.
//
// Deprecated: You can set material.LabelStyle.MaxLines to achieve truncation
// without this type. This type has been reimplemented to do that internally.
type TruncatingLabelStyle material.LabelStyle

// Layout renders the label into the provided context.
func (t TruncatingLabelStyle) Layout(gtx layout.Context) layout.Dimensions {
	t.MaxLines = 1
	return ((material.LabelStyle)(t)).Layout(gtx)
}
