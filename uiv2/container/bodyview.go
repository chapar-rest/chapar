package container

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

// formatInset keeps the floating Format button clear of the editor's
// scrollbars (14 px) with a little room to spare.
const formatInset = 22

// JSONBodyEditor lays out ed, an editor holding a JSON body, with a Format
// button floating in its bottom-right corner, where it stays in reach
// however long the body is.
func JSONBodyEditor(id string, ed *ui.Editor, deps Deps) ui.View {
	return ui.Column(
		ui.ViewOf(ed).Grow(1),
		ui.Button(id+"-format", ui.Text("Format")).
			IconStart(icons.WandSparkles).
			Tooltip("Indent the JSON body; {{variables}} are kept").
			OnClick(func() { FormatJSONEditor(ed, deps) }).
			Float(ui.CornerBottomRight, formatInset, formatInset),
	).Grow(1)
}

// FormatJSONEditor indents the JSON in ed as one edit the user can undo. A
// body that is not valid JSON is left alone and the reason is shown.
func FormatJSONEditor(ed *ui.Editor, deps Deps) {
	formatted, err := FormatJSONBody(string(ed.Bytes()), EditorIndent())
	if err != nil {
		deps.Toast(err.Error())
		return
	}
	ed.SetText(formatted)
}
