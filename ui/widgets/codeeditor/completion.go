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

	active      bool
	ctx         gvcode.CompletionContext
	insertStart int
	candidates  []gvcode.CompletionCandidate
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
	if idx < 0 || idx >= len(tc.candidates) {
		return
	}

	candidate := tc.candidates[idx]
	editStart := tc.insertStart + 2
	editEnd := tc.ctx.Position.Runes
	tc.editor.SetCaret(editStart, editEnd)
	tc.editor.Insert(candidate.Label + "}}")
	tc.Cancel()
	if tc.onConfirm != nil {
		tc.onConfirm()
	}
}

func (tc *templateCompletion) Cancel() {
	tc.active = false
	tc.candidates = tc.candidates[:0]
	tc.insertStart = 0
	if tc.popup != nil {
		tc.popup.Reset()
	}
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
	if !tc.IsActive() {
		// Always run popup layout when inactive so stale key handlers are removed.
		return tc.popup.Layout(gtx, nil)
	}
	return tc.popup.Layout(gtx, tc.candidates)
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
