package gvcode

import (
	"bufio"
	"io"
	"sort"
	"strings"

	lt "github.com/oligo/gvcode/internal/layout"
)

// find a paragraph by rune index.
func (e *textView) findParagraph(runeIdx int) lt.Paragraph {
	idx := sort.Search(len(e.layouter.Paragraphs), func(i int) bool {
		rng := e.layouter.Paragraphs[i]
		return rng.RuneOff+rng.Runes > runeIdx
	})

	// No exsiting paragraph found.
	if idx == len(e.layouter.Paragraphs) {
		return lt.Paragraph{}
	}

	return e.layouter.Paragraphs[idx]
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

// AdjustIndentation indent or unindent each of the selected non-empty lines with
// one tab(soft tab or hard tab).
func (e *textView) AdjustIndentation(textOflines []byte, unindent bool) int {
	indentation := "\t"
	if e.SoftTab {
		indentation = strings.Repeat(" ", e.TabWidth)
	}

	lineReader := bufio.NewReader(strings.NewReader(string(textOflines)))
	newLines := strings.Builder{}

	for {
		line, err := lineReader.ReadBytes('\n')
		if err == io.EOF && len(line) == 0 {
			break
		}

		// empty line with only the trailing line break
		if len(line) == 1 {
			newLines.Write(line)
			continue
		}

		if unindent {
			if strings.HasPrefix(string(line), indentation) {
				newLines.WriteString(strings.TrimPrefix(string(line), indentation))
			} else {
				// TODO: trim any leading whitespaces when no indentation found.
				newLines.Write(line)
			}
		} else {
			newLines.WriteString(indentation + string(line))
		}
	}

	start, end := e.SelectedLineRange()
	return e.Replace(start, end, newLines.String())
}
