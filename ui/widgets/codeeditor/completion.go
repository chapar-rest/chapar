package codeeditor

import (
	"image"
	"strings"
	"unicode/utf8"

	"gioui.org/layout"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/addons/completion"
)

// templateCompletion implements gvcode.Completion for {{variable}} placeholders.
type templateCompletion struct {
	editor    *gvcode.Editor
	popup     *completion.CompletionPopup
	completor *envVariableCompletor
	onConfirm func()

	active       bool
	suppressNext bool
	pendingConfirm int
	ctx            gvcode.CompletionContext
	insertStart    int
	candidates     []gvcode.CompletionCandidate
}

func newTemplateCompletion(editor *gvcode.Editor, popup *completion.CompletionPopup, completor *envVariableCompletor) *templateCompletion {
	return &templateCompletion{
		editor:    editor,
		popup:     popup,
		completor: completor,
	}
}

func (tc *templateCompletion) AddCompletor(_ gvcode.Completor, _ gvcode.CompletionPopup) error {
	return nil
}

func (tc *templateCompletion) OnText(ctx gvcode.CompletionContext) {
	if tc.suppressNext {
		tc.suppressNext = false
		return
	}

	if ctx.Input != "" && isTemplateTerminatingInput(ctx.Input) {
		tc.Cancel()
		return
	}

	insertStart, partial, ok := templateContextBeforeCaret(tc.editor.Text(), ctx.Position.Runes)
	if !ok {
		tc.Cancel()
		return
	}

	tc.active = true
	tc.ctx = ctx
	tc.insertStart = insertStart
	tc.candidates = tc.completor.FilterAndRank(partial, tc.completor.Suggest(ctx))
	if len(tc.candidates) == 0 {
		tc.Cancel()
	}
}

func (tc *templateCompletion) OnConfirm(idx int) {
	tc.pendingConfirm = idx
}

func (tc *templateCompletion) applyConfirm(idx int) {
	if idx < 0 || idx >= len(tc.candidates) {
		return
	}

	candidate := tc.candidates[idx]
	editStart := tc.insertStart + 2
	editEnd := tc.ctx.Position.Runes
	suffix := templateClosingSuffix(tc.editor.Text(), editEnd)
	insertText := candidate.Label + suffix
	tc.editor.SetCaret(editStart, editEnd)
	tc.editor.Insert(insertText)

	caret, _ := tc.editor.Selection()
	caret = advanceCaretPastClosingBraces(tc.editor.Text(), caret)
	tc.editor.SetCaret(caret, caret)

	tc.suppressNext = true
	tc.Cancel()
	if tc.onConfirm != nil {
		tc.onConfirm()
	}
}

func (tc *templateCompletion) Cancel() {
	tc.active = false
	tc.candidates = tc.candidates[:0]
	tc.insertStart = 0
}

func (tc *templateCompletion) IsActive() bool {
	return tc.active && len(tc.candidates) > 0
}

func (tc *templateCompletion) Offset() image.Point {
	return tc.ctx.Coords
}

func (tc *templateCompletion) Layout(gtx layout.Context) layout.Dimensions {
	if tc.popup == nil {
		return layout.Dimensions{}
	}

	var items []gvcode.CompletionCandidate
	if tc.IsActive() {
		items = tc.candidates
	}

	dims := tc.popup.Layout(gtx, items)
	if tc.pendingConfirm >= 0 {
		idx := tc.pendingConfirm
		tc.pendingConfirm = -1
		tc.applyConfirm(idx)
		return tc.popup.Layout(gtx, nil)
	}
	return dims
}

type envVariableCompletor struct {
	editor *gvcode.Editor
	list   VariableLister
}

func (c *envVariableCompletor) Trigger() gvcode.Trigger {
	return gvcode.Trigger{
		Characters: []string{"{"},
		String:     true,
	}
}

func (c *envVariableCompletor) Suggest(ctx gvcode.CompletionContext) []gvcode.CompletionCandidate {
	insertStart, _, ok := templateContextBeforeCaret(c.editor.Text(), ctx.Position.Runes)
	if !ok || c.list == nil {
		return nil
	}

	vars := c.list()
	candidates := make([]gvcode.CompletionCandidate, 0, len(vars))
	for _, v := range vars {
		candidates = append(candidates, gvcode.CompletionCandidate{
			Label:       v.Name,
			Description: truncateVariableValue(v.Value),
			Kind:        v.Kind,
			TextEdit:    gvcode.NewTextEditWithRuneOffset(v.Name+"}}", insertStart+2, ctx.Position.Runes),
		})
	}
	return candidates
}

func (c *envVariableCompletor) FilterAndRank(pattern string, candidates []gvcode.CompletionCandidate) []gvcode.CompletionCandidate {
	if pattern == "" {
		return candidates
	}

	filtered := make([]gvcode.CompletionCandidate, 0, len(candidates))
	lower := strings.ToLower(pattern)
	for _, cand := range candidates {
		if strings.HasPrefix(strings.ToLower(cand.Label), lower) {
			filtered = append(filtered, cand)
		}
	}
	return filtered
}

func templateContextBeforeCaret(text string, runeOff int) (bracesStart int, partial string, ok bool) {
	if runeOff < 2 {
		return 0, "", false
	}

	runes := []rune(text)
	i := runeOff - 1
	for i >= 0 && isTemplateNameRune(runes[i]) {
		i--
	}

	nameStart := i + 1
	partial = string(runes[nameStart:runeOff])

	bracesStart = nameStart - 2
	if bracesStart < 0 || runes[bracesStart] != '{' || runes[bracesStart+1] != '{' {
		return 0, "", false
	}

	return bracesStart, partial, true
}

// templateClosingSuffix returns any missing "}}" characters already present
// after runeOff, for example when auto-close brackets inserted them.
func templateClosingSuffix(text string, runeOff int) string {
	runes := []rune(text)
	var suffix strings.Builder
	for _, want := range "}}" {
		if runeOff < len(runes) && runes[runeOff] == want {
			runeOff++
			continue
		}
		suffix.WriteRune(want)
	}
	return suffix.String()
}

func advanceCaretPastClosingBraces(text string, runeOff int) int {
	runes := []rune(text)
	for runeOff < len(runes) && runes[runeOff] == '}' {
		runeOff++
	}
	return runeOff
}

func isTemplateNameRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' || r == '-' || r == '$'
}

func isTemplateTerminatingInput(input string) bool {
	if input == "" {
		return false
	}

	switch []rune(input)[0] {
	case '}', ' ', '\n', '\t', '(', ')', ',', ';', '"', '\'':
		return true
	default:
		return false
	}
}

func truncateVariableValue(value string) string {
	const maxLen = 48
	if utf8.RuneCountInString(value) <= maxLen {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxLen]) + "..."
}
