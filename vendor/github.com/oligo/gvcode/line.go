package gvcode

import (
	"sort"

	lt "github.com/oligo/gvcode/internal/layout"
)

// paragraphOfCaret returns the paragraph that the carent is in.
func (e *textView) paragraphOfCaret(caret caretPos) lt.Paragraph {
	if len(e.layouter.Paragraphs) <= 0 {
		return lt.Paragraph{}
	}

	caretStart := e.closestToRune(caret.start)

	lineIdx := sort.Search(len(e.layouter.Paragraphs), func(i int) bool {
		rng := e.layouter.Paragraphs[i]
		return rng.EndY >= caretStart.Y
	})

	// No exsiting paragraph found.
	if lineIdx == len(e.layouter.Paragraphs) {
		return lt.Paragraph{}
	}

	return e.layouter.Paragraphs[lineIdx]
}

// CurrentLine returns the start and end rune index of the current paragraph
// that the caret is in.
func (e *textView) CurrentLine() (start, end int) {
	p := e.paragraphOfCaret(e.caret)
	if p == (lt.Paragraph{}) {
		return
	}

	return p.RuneOff, p.RuneOff + p.Runes
}

// SelectedLine returns the text of the current line the caret is in.
func (e *textView) SelectedLine(buf []byte) []byte {
	paragraph := e.paragraphOfCaret(e.caret)
	if paragraph == (lt.Paragraph{}) {
		return buf[:0]
	}

	startOff := e.src.RuneOffset(paragraph.RuneOff)
	endOff := e.src.RuneOffset(paragraph.RuneOff + paragraph.Runes)

	if cap(buf) < endOff-startOff {
		buf = make([]byte, endOff-startOff)
	}
	buf = buf[:endOff-startOff]
	n, _ := e.src.ReadAt(buf, int64(startOff))
	return buf[:n]
}
