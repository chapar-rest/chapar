package gvcode

import (
	"bufio"
	"io"
	"slices"
	"strings"
)

type autoIndenter struct {
	*Editor
}

// IndentMultiLines indent or dedent each of the selected non-empty lines with
// one tab(soft tab or hard tab). If there is now selection, the current line is
// indented or dedented.
func (e *autoIndenter) IndentMultiLines(dedent bool) int {
	e.scratch = e.text.SelectedLineText(e.scratch)
	if len(e.scratch) == 0 {
		return 0
	}

	lineReader := bufio.NewReader(strings.NewReader(string(e.scratch)))
	newLines := strings.Builder{}

	moves := 0
	caretMoves := 0
	caretStart, caretEnd := e.text.Selection()
	// caret columns in runes
	_, caretCol := e.text.CaretPos()

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
			newLines.WriteString(e.text.Indentation() + string(line))
			moves += len([]rune(e.text.Indentation()))
		}
	}

	start, end := e.text.SelectedLineRange()
	n := e.text.Replace(start, end, newLines.String())

	if moves != 0 {
		// adjust caret positions
		if dedent {
			// When lines are dedented.
			if caretEnd < caretStart {
				e.text.SetCaret(caretStart+moves, caretEnd+caretMoves)
			} else {
				e.text.SetCaret(caretStart+caretMoves, caretEnd+moves)
			}
		} else {
			// When lines are indented, expand the end of the selection.
			if caretEnd > caretStart {
				e.text.SetCaret(caretStart, caretEnd+moves)
			} else {
				e.text.SetCaret(caretStart+moves, caretEnd)
			}
		}
	}

	return n
}

func (e autoIndenter) dedentLine(line string) string {
	level := 0
	spaces := 0
	off := 0
	for i, r := range line {
		if r == '\t' {
			spaces = 0
			off = i
			level++
		} else if r == ' ' {
			if spaces == 0 || spaces == e.text.TabWidth {
				off = i
				if spaces == e.text.TabWidth {
					spaces = 0
				}
			}
			spaces++
			if spaces == e.text.TabWidth {
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

// IndentOnBreak insert a line break at the the current caret position, and if there is any indentation
// of the previous line, it indent the new inserted line with the same size. Furthermore, if the newline
// if between a pair of brackets, it also insert indented lines between them.
//
// This is mainly used as the line break handler when Enter or Return is pressed.
func (e *autoIndenter) IndentOnBreak(s string) bool {
	start, end := e.text.Selection()
	if s != "\n" || start != end {
		return e.Insert(s) > 0
	}

	// Find the previous paragraph.
	p := e.text.SelectedLineText(e.scratch)
	if len(p) == 0 {
		return e.Insert(s) > 0
	}

	indentation := e.text.Indentation()
	indents := 0
	spaces := 0
	for _, r := range string(p) {
		if r == '\t' {
			indents++
		} else if r == ' ' {
			spaces++
			if spaces == e.text.TabWidth {
				indents++
				spaces = 0
				continue
			}
		} else {
			// other chars
			break
		}
	}

	if indents > 0 {
		s = s + strings.Repeat(indentation, indents)
	}
	changed := e.Insert(s) > 0
	if !changed {
		return false
	}
	// Check if the caret is between a pair of brackets. If so we insert one more
	// indented empty line between the pair of brackets.
	return e.indentInsideBrackets(indents)
}

// indentInsideBrackets checks if the caret is between two adjacent brackets pairs and insert
// indented lines between them.
func (e *autoIndenter) indentInsideBrackets(indents int) bool {
	start, end := e.text.Selection()
	if start <= 0 || start != end {
		return false
	}

	indentation := e.text.Indentation()
	moves := indents * len([]rune(indentation))

	leftRune, err1 := e.buffer.ReadRuneAt(start - 2 - moves) // offset to index
	rightRune, err2 := e.buffer.ReadRuneAt(min(start, e.text.Len()))

	if err1 != nil || err2 != nil {
		return false
	}

	insideBrackets := rightRune == e.text.BracketPairs[leftRune]
	if insideBrackets {
		// move to the left side of the line break.
		e.text.MoveCaret(-moves, -moves)
		// Add one more line and indent one more level.
		changed := e.Insert(strings.Repeat(indentation, indents+1)+"\n") != 0
		if changed {
			e.text.MoveCaret(-1, -1)
			return true
		}
	}

	return false
}

// func (e *autoIndentHandler) dedentRightBrackets(ke key.EditEvent) bool {
// 	opening, ok := rtlBracketPairs[ke.Text]
// 	if !ok {
// 		return false
// 	}
