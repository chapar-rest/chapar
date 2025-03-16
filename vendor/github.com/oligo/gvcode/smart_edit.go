package gvcode

import (
	"bufio"
	"bytes"
	"io"
	"maps"
	"strings"

	"gioui.org/io/key"
)

// Bracket pairs
var ltrBracketPairs = map[string]string{
	"(": ")",
	"{": "}",
	"[": "]",
}
var rtlBracketPairs = map[string]string{
	"(": ")",
	"{": "}",
	"[": "]",
}
var quotePairs = map[string]string{
	"'":  "'",
	"\"": "\"",
	"`":  "`",
}

var autoCompletablePairs = mergeMaps(ltrBracketPairs, quotePairs)

func (e *Editor) autoCompleteTextPair(ke key.EditEvent) bool {
	closing, ok := autoCompletablePairs[ke.Text]
	if !ok {
		return false
	}

	e.scrollCaret = true
	e.scroller.Stop()
	e.replace(ke.Range.Start, ke.Range.End, ke.Text+closing)
	e.text.MoveCaret(-len([]rune(closing)), -len([]rune(closing)))
	return true
}

func mergeMaps(sources ...map[string]string) map[string]string {
	dest := make(map[string]string)
	for _, src := range sources {
		maps.Copy(dest, src)
	}

	return dest
}

// AdjustIndentation indent or unindent each of the selected non-empty lines with
// one tab(soft tab or hard tab).
func (e *Editor) adjustIndentation(textOflines []byte, unindent bool) int {
	indentation := e.text.Indentation()
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

	start, end := e.text.SelectedLineRange()
	return e.text.Replace(start, end, newLines.String())
}

// breakAndIndent insert a line break at the the current caret position, and if there is any indentation
// of the previous line, it indent the new inserted line with the same size.
//
// This is part of the line break handler when Enter or Return is pressed.
func (e *Editor) breakAndIndent(s string) (bool, int) {
	start, end := e.text.Selection()
	if s != "\n" || start != end {
		return e.Insert(s) > 0, 0
	}

	// Find the previous paragraph.
	p := e.text.SelectedLineText(e.scratch)
	if len(p) == 0 {
		return e.Insert(s) > 0, 0
	}

	indentation := e.text.Indentation()
	indents := 0
	for {
		if !bytes.HasPrefix(p, []byte(indentation)) {
			break
		}

		indents++
		p = p[len(indentation):]
	}

	return e.Insert(s+strings.Repeat(indentation, indents)) > 0, indents
}

// indentInsideBrackets checks if the caret is between two adjascent brackets pairs and insert
// indented lines between them. 
// 
// This is part of the line break handler when Enter or Return is pressed.
func (e *Editor) indentInsideBrackets(indents int) {
	start, end := e.text.Selection()
	if start <= 0 || start != end {
		return
	}

	indentation := e.text.Indentation()
	moves := indents * len([]rune(indentation))

	leftRune, err1 := e.buffer.ReadRuneAt(start - 2 - moves) // offset to index
	rightRune, err2 := e.buffer.ReadRuneAt(min(start, e.text.Len()))

	if err1 != nil && err2 != nil {
		return
	}

	insideBrackets := string(rightRune) == ltrBracketPairs[string(leftRune)]
	if insideBrackets {
		// move to the left side of the line break.
		e.text.MoveCaret(-moves, -moves)
		// Add one more line and indent one more level.
		changed := e.Insert(strings.Repeat(indentation, indents+1)+"\n") != 0
		if changed {
			e.text.MoveCaret(-1, -1)
		}
	}

}
