package textview

import (
	"bufio"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

// IndentLines indent or dedent each of the selected non-empty lines with
// one tab(soft tab or hard tab). If there is no selection, the current line is
// indented or dedented.
func (e *TextView) IndentLines(dedent bool) int {
	// 1. normal case: insert a TAB forward.
	if selectedLines := e.selectedParagraphs(); !dedent && len(selectedLines) <= 1 {
		// expand soft tab.
		start, end := e.Selection()
		moves := e.Replace(start, end, e.expandTab(start, end, "\t"))
		if start != end {
			e.ClearSelection()
			e.MoveCaret(moves, moves)
		}
		return moves
	}

	// 2. Otherwise, indent or dedent all the selected lines.
	var linesStart, linesEnd int
	e.lineBuf, linesStart, linesEnd = e.SelectedLineText(e.lineBuf)
	if len(e.lineBuf) == 0 {
		return 0
	}

	lineReader := bufio.NewReader(strings.NewReader(string(e.lineBuf)))
	newLines := strings.Builder{}
	moves := 0
	caretMoves := 0
	caretStart, caretEnd := e.Selection()
	// caret columns in runes
	_, caretCol := e.CaretPos()

	for i := 0; ; i++ {
		line, err := lineReader.ReadBytes('\n')
		if err == io.EOF && len(line) == 0 {
			break
		}

		// empty line with only the trailing line break
		if len(line) == 1 {
			newLines.Write(line)
			continue
		}

		if dedent {
			newLine := e.dedentLine(string(line))
			newLines.WriteString(newLine)
			delta := len([]rune(newLine)) - len([]rune(string(line)))
			moves += delta
			if caretEnd > caretStart {
				caretMoves = max(delta, -caretCol)
			} else {
				// capture the last line indent moves
				caretMoves = delta
			}

		} else {
			newLines.WriteString(e.Indentation() + string(line))
			moves += len([]rune(e.Indentation()))
		}
	}

	var inserted int
	if newLines.String() != string(e.lineBuf) {
		inserted = e.Replace(linesStart, linesEnd, newLines.String())
	}

	if moves != 0 {
		// adjust caret positions
		if dedent {
			// When lines are dedented.
			if caretEnd < caretStart {
				e.SetCaret(caretStart+moves, caretEnd+caretMoves)
			} else {
				e.SetCaret(caretStart+caretMoves, caretEnd+moves)
			}
		} else {
			// When lines are indented, expand the end of the selection.
			if caretEnd > caretStart {
				e.SetCaret(caretStart, caretEnd+moves)
			} else {
				e.SetCaret(caretStart+moves, caretEnd)
			}
		}
	}

	return inserted
}

func (e *TextView) dedentLine(line string) string {
	level := 0
	spaces := 0
	off := 0
	for i, r := range line {
		if r == '\t' {
			spaces = 0
			off = i
			level++
		} else if r == ' ' {
			if spaces == 0 || spaces == e.TabWidth {
				off = i
				if spaces == e.TabWidth {
					spaces = 0
				}
			}
			spaces++
			if spaces == e.TabWidth {
				level++
				continue
			}
		} else {
			// other chars
			break
		}
	}

	if spaces > 0 {
		// trim left over spaces first
		return string(slices.Delete([]rune(line), off, off+spaces))
	} else if level > 0 {
		// try to delete a single tab just before the non-spaces text.
		return string(slices.Delete([]rune(line), off, off+1))
	}

	return line
}

func checkIndentLevel(line []byte, tabWidth int) int {
	indents := 0
	spaces := 0
	for _, r := range string(line) {
		if r == '\t' {
			indents++
		} else if r == ' ' {
			spaces++
			if spaces == tabWidth {
				indents++
				spaces = 0
				continue
			}
		} else {
			// other chars
			break
		}
	}

	return indents
}

// IndentOnBreak insert a line break at the the current caret position, and if there is any indentation
// of the previous line, it indent the new inserted line with the same size. Furthermore, if the newline
// if between a pair of brackets, it also insert indented lines between them.
//
// This is mainly used as the line break handler when Enter or Return is pressed.
func (e *TextView) IndentOnBreak(s string) int {
	var lineStart, lineEnd int
	e.lineBuf, lineStart, lineEnd = e.SelectedLineText(e.lineBuf)

	start, end := e.Selection()
	indents := checkIndentLevel(e.lineBuf, e.TabWidth)
	buf := &strings.Builder{}
	adjust := 0

	// 1. normal case:
	buf.WriteString(s)
	buf.WriteString(strings.Repeat(e.Indentation(), indents))

	// 2. check if we are after inside a brackets pair.
	leftBracket, rightBracket := e.NearestMatchingBrackets()
	inBrackets := leftBracket >= 0 && rightBracket > leftBracket &&
		lineStart <= leftBracket && leftBracket < lineEnd && end <= rightBracket
	if inBrackets {
		// Inside of a pair of brackets, add one more level of indents.
		buf.WriteString(e.Indentation())

		// 3 check if the right rune happens to be a right bracket
		if rightBracket <= lineEnd && end == rightBracket {
			s2 := s + strings.Repeat(e.Indentation(), indents)
			buf.WriteString(s2)
			adjust += utf8.RuneCountInString(s2)
		}

	}

	moves := e.Replace(start, end, buf.String())
	if start != end {
		// if there is a seletion, clear the selection.
		e.ClearSelection()
		adjust -= moves
	}

	// get the updated selection.
	start, end = e.Selection()
	e.SetCaret(start-adjust, end-adjust)

	return moves
}

// func (e *autoIndentHandler) dedentRightBrackets(ke key.EditEvent) bool {
// 	opening, ok := rtlBracketPairs[ke.Text]
// 	if !ok {
// 		return false
// 	}
