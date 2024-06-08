// SPDX-License-Identifier: Unlicense OR MIT
// Copied from: gioui material/radiobutton.go with some modifications
package widgets

import (
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type RadioButtonStyle struct {
	checkable
	Key   string
	Group *widget.Enum
}

// RadioButton returns a RadioButton with a label. The key specifies
// the value for the Enum.
func RadioButton(th *material.Theme, group *widget.Enum, key, label string) RadioButtonStyle {
	r := RadioButtonStyle{
		Group: group,
		checkable: checkable{
			Label: label,

			Color:              th.Palette.Fg,
			IconColor:          th.Palette.ContrastBg,
			TextSize:           th.TextSize * 14.0 / 16.0,
			Size:               26,
			shaper:             th.Shaper,
			checkedStateIcon:   th.Icon.RadioChecked,
			uncheckedStateIcon: th.Icon.RadioUnchecked,
		},
		Key: key,
	}
	r.checkable.Font.Typeface = th.Face
	return r
}

// Layout updates enum and displays the radio button.
func (r RadioButtonStyle) Layout(gtx layout.Context) layout.Dimensions {
	r.Group.Update(gtx)
	return r.Group.Layout(gtx, r.Key, func(gtx layout.Context) layout.Dimensions {
		semantic.RadioButton.Add(gtx.Ops)
		return r.layout(gtx, r.Group.Value == r.Key)
	})
}
