package container

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

// floatInset is how far the buttons floating over a body editor sit from its
// edges.
const floatInset = 11

// JSONBodyEditor lays out ed, an editor holding a JSON body, with a Format
// button floating in its bottom-right corner, where it stays in reach
// however long the body is. topRight, when not nil, floats in the top-right
// corner, for an action such as loading an example body.
func JSONBodyEditor(id string, ed *ui.Editor, deps Deps, topRight *ui.Node) ui.View {
	var top ui.View
	if topRight != nil {
		top = topRight.Float(ui.CornerTopRight, floatInset, floatInset)
	}
	return ui.Column(
		ui.ViewOf(ed).Grow(1),
		top,
		ui.Button(id+"-format", ui.Text("Format")).
			IconStart(icons.WandSparkles).
			Tooltip("Indent the JSON body; {{variables}} are kept").
			OnClick(func() { FormatJSONEditor(ed, deps) }).
			Float(ui.CornerBottomRight, floatInset, floatInset),
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
