package syntax

import (
	"sort"

	"github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/internal/painter"
)

type Token struct {
	// offset of the start rune in the document.
	Start, End int
	// Scope registered in the color scheme.
	Scope StyleScope
}

type TextTokens struct {
	tokens      []TokenStyle
	colorScheme *ColorScheme
	splitter    lineSplitter
}

func NewTextTokens(scheme *ColorScheme) *TextTokens {
	return &TextTokens{
		colorScheme: scheme,
	}
}

// Clear the tokens for reuse.
func (t *TextTokens) Clear() {
	t.tokens = t.tokens[:0]
}

// Set adds all the tokens, replacing the existing ones.
// Caller should insures the tokens are sorted by the range in ascending order .
func (t *TextTokens) Set(tokens ...Token) {
	t.Clear()
	for _, token := range tokens {
		t.add(token.Scope, token.Start, token.End)
	}
}

func (t *TextTokens) add(scope StyleScope, start, end int) {
	style := t.colorScheme.GetTokenStyle(scope)
	if style == 0 {
		return
	}

	t.tokens = append(t.tokens, TokenStyle{
		Start: start,
		End:   end,
		Style: style,
	})
}

func (t *TextTokens) GetColor(colorID int) color.Color {
	return t.colorScheme.GetColor(colorID)
}

// Query tokens for rune range. start and end are in runes. start is inclusive
// and end is exclusive. This method assumes the tokens are sorted by start or end
// in ascending order.
func (t *TextTokens) QueryRange(start, end int) []TokenStyle {
	if len(t.tokens) == 0 || start >= end {
		return nil
	}

	// Find the index of the first token whose End is greater than start.
	// Tokens before this index cannot overlap because they end too early.
	firstIdx := sort.Search(len(t.tokens), func(i int) bool {
		return t.tokens[i].End > start
	})

	if firstIdx == len(t.tokens) {
		// All tokens end before start, so no overlap.
		return nil
	}

	var result []TokenStyle
	for i := firstIdx; i < len(t.tokens); i++ {
		token := t.tokens[i]
		if token.Start < end {
			result = append(result, token)
		} else {
			// This token starts at or after end, no overlap.
			// Since tokens are sorted by Start, we can break early.
			break
		}
	}
	return result
}

// Split implements painter.LineSplitter
func (t *TextTokens) Split(line *layout.Line, runs *[]painter.RenderRun) {
	t.splitter.Split(line, t, runs)
}
