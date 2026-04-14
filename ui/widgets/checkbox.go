package widgets

import (
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type CheckBoxStyle struct {
	checkable
	CheckBox *widget.Bool
}

func CheckBox(th *chapartheme.Theme, checkBox *widget.Bool, label string) CheckBoxStyle {
	c := CheckBoxStyle{
		CheckBox: checkBox,
		checkable: checkable{
			Label:              label,
			Color:              th.Fg,
			IconColor:          th.CheckBoxColor,
			TextSize:           th.TextSize * 12.0 / 14.0,
			Size:               24,
			shaper:             th.Shaper,
			checkedStateIcon:   th.Icon.CheckBoxChecked,
			uncheckedStateIcon: th.Icon.CheckBoxUnchecked,
		},
	}
	c.Font.Typeface = th.Face
	return c
}

// Layout updates the checkBox and displays it.
func (c CheckBoxStyle) Layout(gtx layout.Context) layout.Dimensions {
	return c.CheckBox.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.CheckBox.Add(gtx.Ops)
		return c.layout(gtx, c.CheckBox.Value)
	})
}
