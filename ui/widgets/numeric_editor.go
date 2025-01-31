package widgets

import (
	"strconv"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type NumericEditor struct {
	widget.Editor
}

func (n *NumericEditor) Value() int {
	v, _ := strconv.Atoi(n.Text())
	return v
}

func (n *NumericEditor) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	for {
		event, ok := n.Update(gtx)
		if !ok {
			break
		}

		switch event.(type) {
		// on change event
		case widget.ChangeEvent:
			if n.Text() != "" {
				if _, err := strconv.Atoi(n.Text()); err != nil {
					n.SetText(n.Text()[:len(n.Text())-1])
				}
			}
		}
	}

	editor := material.Editor(theme.Material(), &n.Editor, "0")
	editor.SelectionColor = theme.TextSelectionColor
	return editor.Layout(gtx)
}
