package keys

import (
	"gioui.org/layout"
	"gioui.org/widget"
)

func OnEditorChange(gtx layout.Context, editor *widget.Editor, callback func()) {
	for {
		event, ok := editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.ChangeEvent); ok {
			callback()
		}
	}
}
