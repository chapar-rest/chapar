package gvcode

import (
	"sort"
	"strings"

	lt "github.com/oligo/gvcode/internal/layout"
)

// find a paragraph by rune index, returning the line number(starting from zero)
// and the paragraph itself.
func (e *textView) FindParagraph(runeIdx int) (int, lt.Paragraph) {
	idx := sort.Search(len(e.layouter.Paragraphs), func(i int) bool {
		rng := e.layouter.Paragraphs[i]
		return rng.RuneOff+rng.Runes > runeIdx
	})

	// No exsiting paragraph found.
	if idx == len(e.layouter.Paragraphs) {
		return 0, lt.Paragraph{}
	}

	return idx, e.layouter.Paragraphs[idx]
}

// selectedParagraphs returns the paragraphs that the carent selection covers.
// If there's no selection, it returns the paragraph that the caret is in.
func (e *textView) selectedParagraphs() []lt.Paragraph {
	if len(e.layouter.Paragraphs) <= 0 {
		return nil
	}

	selections := make([]lt.Paragraph, 0)

	caretStart := min(e.caret.start, e.caret.end)
	caretEnd := max(e.caret.start, e.caret.end)

	startIdx := sort.Search(len(e.layouter.Paragraphs), func(i int) bool {
		rng := e.layouter.Paragraphs[i]
		return rng.EndY >= e.closestToRune(caretStart).Y
	})

	// No exsiting paragraph found.
	if startIdx == len(e.layouter.Paragraphs) {
		return selections
	}
	selections = append(selections, e.layouter.Paragraphs[startIdx])

	if caretStart != caretEnd {
		endIdx := sort.Search(len(e.layouter.Paragraphs), func(i int) bool {
			rng := e.layouter.Paragraphs[i]
			return rng.EndY >= e.closestToRune(caretEnd).Y
		})

		if endIdx == len(e.layouter.Paragraphs) {
			return selections
		}

		for i := startIdx + 1; i <= endIdx; i++ {
			p := e.layouter.Paragraphs[i]
			if i == endIdx && p.RuneOff == caretEnd {
				// skip the last empty-selection line as it indicates we are at the end
				// of the previous line.
				break
			}
			selections = append(selections, e.layouter.Paragraphs[i])
		}
	}

	return selections

}

// SelectedLineRange returns the start and end rune index of the paragraphs selected by the caret.
// If there is no selection, the range of current paragraph the caret is in is returned.
func (e *textView) SelectedLineRange() (start, end int) {
	paragraphs := e.selectedParagraphs()
	if len(paragraphs) == 0 {
		return
	}

	last := paragraphs[len(paragraphs)-1]
	return paragraphs[0].RuneOff, last.RuneOff + last.Runes
}

// SelectedLine returns the text of the selected lines. An empty selection is treated
// as a single line selection.
func (e *textView) SelectedLineText(buf []byte) []byte {
	paragraphs := e.selectedParagraphs()
	if len(paragraphs) == 0 {
		return buf[:0]
	}

	start := paragraphs[0].RuneOff
	end := paragraphs[len(paragraphs)-1].RuneOff + paragraphs[len(paragraphs)-1].Runes

	startOff := e.src.RuneOffset(start)
	endOff := e.src.RuneOffset(end)

	if cap(buf) < endOff-startOff {
		buf = make([]byte, endOff-startOff)
	}
	buf = buf[:endOff-startOff]
	n, _ := e.src.ReadAt(buf, int64(startOff))
	return buf[:n]
}

// partialLineSelected checks if the current selection is a partial single line.
func (e *textView) PartialLineSelected() bool {
	if e.caret.start == e.caret.end {
		return false
	}

	paragraphs := e.selectedParagraphs()
	if len(paragraphs) > 1 {
		return false
	}

	caretStart := min(e.caret.start, e.caret.end)
	caretEnd := max(e.caret.start, e.caret.end)
	p := paragraphs[0]

	if p.RuneOff != caretStart {
		return true
	}

	lastRune, err := e.src.ReadRuneAt(p.RuneOff + p.Runes - 1)
	if err != nil {
		// TODO: how to handle the read error?
	}

	if lastRune == '\n' {
		return p.RuneOff+p.Runes != caretEnd+1
	} else {
		return p.RuneOff+p.Runes != caretEnd
	}
}

// expandTab tries to expand tab character to spaces while respecting tab stops.
// If s is a single tab character and the editor is configured to use soft tab,
// the tab is expanded with spaces, also tab stop is accounted when calculating
// space number.
func (e *textView) ExpandTab(start, end int, s string) string {
	if !e.SoftTab || s != "\t" {
		return s
	}

	if start > end {
		start = end
	}

	_, p := e.FindParagraph(start)
	if p == (lt.Paragraph{}) {
		return strings.Repeat(" ", e.TabWidth)
	}

	advance := start - p.RuneOff
	nextTabStop := (advance/e.TabWidth + 1) * e.TabWidth
	spaces := nextTabStop - advance

	return strings.Repeat(" ", spaces)
}

// Indentation returns the text sequence used to indent the lines(paragraphs).
func (e *textView) Indentation() string {
	indentation := "\t"
	if e.SoftTab {
		indentation = strings.Repeat(" ", e.TabWidth)
	}
	return indentation
}
