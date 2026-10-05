package container

import "github.com/mirzakhany/yoga/ui"

// MutedParagraph is body text in ForegroundMuted that wraps to its pane's width.
func MutedParagraph(s string) *ui.Node {
	return ui.Paragraph(s).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
}
