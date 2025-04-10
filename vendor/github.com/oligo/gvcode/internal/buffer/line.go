package buffer

import (
	"slices"
	"unicode/utf8"
)

const (
	lineBreak = '\n'
)

type lineInfo struct {
	// size in runes of the line
	length       int
	hasLineBreak bool
}

type lineOp struct {
	action    action
	runeIndex int
	length    int
	lines     []lineInfo
}

type lineOpStack struct {
	ops []*lineOp
}

// Deprecated: We don't need to calculate precise line range anymore.
//
// lineIndex manages a line index for the text sequence using a hybrid strategy:
//  1. update the index when insert or erase occurs in an incremental manner.
//  2. rebuild the index when undo or redo is triggerred in the PieceTable.
//
// It provides its own undo/redo stack for internal undo and redo. But this is hard to
// align with the undo and redo operation of PieceTable. These two methods may be removed
// in the feture.
type lineIndex struct {
	// Index of the slice saves the continuous line number starting from zero.
	// The value contains the rune length of the line.
	lines []lineInfo
	// undo stack & redo stack used to update the index after the piece range is undone or redone.
	undo lineOpStack
	redo lineOpStack
}

func (li *lineIndex) UpdateOnInsert(runeIndex int, text []byte) {
	li.redo.clear()
	newLines := li.parseLine(text)
	li.applyInsert(runeIndex, newLines)

	op := &lineOp{
		action:    actionInsert,
		runeIndex: runeIndex,
		length:    utf8.RuneCountInString(string(text)),
		lines:     newLines,
	}

	li.undo.push(op)
}

func (li *lineIndex) UpdateOnDelete(runeIndex int, length int) {
	li.redo.clear()
	removedLines := li.applyDelete(runeIndex, length)

	op := &lineOp{
		action:    actionErase,
		runeIndex: runeIndex,
		length:    length,
		lines:     removedLines,
	}
	li.undo.push(op)

}

// Optional. To make the situation simpler, we use Rebuild to build the
// entire index from scratch.
func (li *lineIndex) Undo() {
	src := li.undo
	dest := li.redo

	op := src.pop()
	if op == nil {
		return
	}

	if op.action == actionInsert {
		li.applyDelete(op.runeIndex, op.length)
	} else if op.action == actionErase {
		li.applyInsert(op.runeIndex, op.lines)
	}

	dest.push(op)
}

// Optional. To make the situation simpler, we use Rebuild to build the
// entire index from scratch.
func (li *lineIndex) Redo() {
	src := li.redo
	dest := li.undo

	op := src.pop()
	if op == nil {
		return
	}

	if op.action == actionInsert {
		li.applyInsert(op.runeIndex, op.lines)
	} else if op.action == actionErase {
		li.applyDelete(op.runeIndex, op.length)
	}

	dest.push(op)
}

func (li *lineIndex) Rebuild(pt *PieceTable) {
	li.lines = li.lines[:0]
	for n := pt.pieces.Head(); n != pt.pieces.tail; n = n.next {
		pieceText := pt.getBuf(n.source).getTextByRange(n.byteOff, n.byteLength)
		lines := li.parseLine(pieceText)
		if len(lines) > 0 {
			if len(li.lines) > 0 {
				lastLine := li.lines[len(li.lines)-1]
				if !lastLine.hasLineBreak {
					// merge with lastLine
					lines[0].length += lastLine.length
					li.lines = li.lines[:len(li.lines)-1]
				}
			}

			li.lines = append(li.lines, lines...)
		}
	}
}

func (li *lineIndex) applyInsert(runeIndex int, newLines []lineInfo) {
	if len(newLines) <= 0 {
		return
	}

	var currentRuneCount int
	var insertionIndex int

	// Locate the insertion point in the existing line index
	for i, line := range li.lines {
		if currentRuneCount+line.length > runeIndex {
			insertionIndex = i
			break
		}
		currentRuneCount += line.length
	}

	// Check if we have found a insertion point.
	if currentRuneCount > 0 && insertionIndex == 0 {
		// No insertion found, increase the index to the next.
		insertionIndex = len(li.lines)
	}

	// Split the line at the insertion point if necessary
	if insertionIndex < len(li.lines) {
		line := li.lines[insertionIndex]

		// Prepare for splitting and merging
		splitLeft := runeIndex - currentRuneCount

		if splitLeft <= 0 {
			li.lines = slices.Insert(li.lines, insertionIndex, newLines...)
		} else {
			if len(newLines) == 1 && !newLines[0].hasLineBreak {
				// just merge the new fragment.
				li.lines[insertionIndex].length += newLines[0].length
			} else {
				// Create a left part from the split
				leftPart := lineInfo{length: splitLeft + newLines[0].length, hasLineBreak: newLines[0].hasLineBreak}
				li.lines[insertionIndex] = leftPart
			}

			newLines = newLines[1:]

			var rightPart lineInfo
			if len(newLines) > 0 {
				lastLine := newLines[len(newLines)-1]
				if !lastLine.hasLineBreak {
					rightPart = lineInfo{length: line.length - splitLeft + lastLine.length, hasLineBreak: line.hasLineBreak}
					newLines = newLines[:len(newLines)-1]
				} else {
					rightPart = lineInfo{length: line.length - splitLeft, hasLineBreak: line.hasLineBreak}
				}

				li.lines = slices.Insert(li.lines, insertionIndex+1, rightPart)
			}

			if len(newLines) > 0 {
				li.lines = slices.Insert(li.lines, insertionIndex+1, newLines...)
			}
		}

		return
	}

	// If the last line does not have a line break, merge the first new line with it.
	if len(li.lines) > 0 {
		lastLine := li.lines[len(li.lines)-1]
		if !lastLine.hasLineBreak {
			lastLine.length += newLines[0].length
			lastLine.hasLineBreak = newLines[0].hasLineBreak
			li.lines[len(li.lines)-1] = lastLine
			newLines = newLines[:len(newLines)-1]
		}
	}

	// Append the remaining lines
	li.lines = append(li.lines, newLines...)
}

func (li *lineIndex) applyDelete(runeIndex int, length int) []lineInfo {
	var startLineRuneCount, endLineRuneCount int
	var startIndex, endIndex int
	var removedLines []lineInfo

	// Locate the starting and ending indices of the deletion
	for i, line := range li.lines {
		if startLineRuneCount+line.length > runeIndex {
			startIndex = i
			break
		}
		startLineRuneCount += line.length
	}

	endLineRuneCount = startLineRuneCount
	for i, line := range li.lines[startIndex:] {
		if endLineRuneCount+line.length >= runeIndex+length {
			endIndex = startIndex + i
			break
		}
		endLineRuneCount += line.length
	}

	// If startIndex equals to endIndex, the line is to be splited into 2 or 3 parts,
	// with the second part removed, and the other two or one being kept.
	// And things would get harder when the end is just at the boundary of the line, in
	// which case the current line should be joined with the next line(if there is any).
	if startIndex == endIndex {
		leftPart := runeIndex - startLineRuneCount
		rightPartLen := li.lines[endIndex].length - leftPart - length
		removedLines = append(removedLines, lineInfo{length: length, hasLineBreak: rightPartLen <= 0})

		// split into 2 parts.
		if rightPartLen <= 0 {
			// merge the current line with the next line
			if endIndex+1 < len(li.lines) {
				li.lines[endIndex+1].length += leftPart
				li.lines = append(li.lines[:endIndex], li.lines[endIndex+1])
			} else {
				li.lines[endIndex] = lineInfo{length: leftPart, hasLineBreak: false}
			}
		} else {
			li.lines[endIndex].length = li.lines[endIndex].length - length
		}

		return removedLines

	}

	// Handle the splitting of the ending line
	if endIndex < len(li.lines) {
		endLine := li.lines[endIndex]
		splitRight := (runeIndex + length) - endLineRuneCount
		removed := lineInfo{
			length: splitRight,
		}

		if splitRight < endLine.length {
			rightPart := lineInfo{length: endLine.length - splitRight, hasLineBreak: endLine.hasLineBreak}
			li.lines[endIndex] = rightPart
			removed.hasLineBreak = false
		} else {
			removed.hasLineBreak = endLine.hasLineBreak
			li.lines = slices.Delete(li.lines, endIndex, endIndex+1)
			//li.lines = append(li.lines[:endIndex], li.lines[endIndex+1:]...)
		}

		removedLines = append(removedLines, removed)
	}

	nextLine := endIndex - 1

	for nextLine > startIndex {
		removedLines = append(removedLines, li.lines[nextLine])
		li.lines = slices.Delete(li.lines, nextLine, nextLine+1)
		nextLine--
	}

	// Handle the splitting of the starting line
	//startLine := li.lines[startIndex]
	splitLeft := runeIndex - startLineRuneCount
	removed := lineInfo{
		length:       li.lines[startIndex].length - splitLeft,
		hasLineBreak: li.lines[startIndex].hasLineBreak,
	}
	removedLines = append(removedLines, removed)
	li.lines[startIndex] = lineInfo{length: splitLeft, hasLineBreak: false}
	if splitLeft <= 0 {
		li.lines = slices.Delete(li.lines, startIndex, startIndex+1)
		nextLine--
	}

	// Check if the previous line has line break.
	// Try to merge it with the previous line.
	if nextLine >= 0 && !li.lines[nextLine].hasLineBreak && len(li.lines) > nextLine+1 {
		li.lines[nextLine].length += li.lines[nextLine+1].length
		li.lines[nextLine].hasLineBreak = li.lines[nextLine+1].hasLineBreak
		li.lines = slices.Delete(li.lines, nextLine+1, nextLine+2)
	}

	return removedLines
}

func (li *lineIndex) parseLine(text []byte) []lineInfo {
	var lines []lineInfo

	n := 0
	for _, c := range string(text) {
		n++
		if c == lineBreak {
			lines = append(lines, lineInfo{length: n, hasLineBreak: true})
			n = 0
		}
	}

	// The remaining bytes that don't end with a line break.
	if n > 0 {
		lines = append(lines, lineInfo{length: n})
	}

	return lines
}

func (li *lineIndex) Lines() []lineInfo {
	return li.lines
}

func (s *lineOpStack) push(op *lineOp) {
	s.ops = append(s.ops, op)
}

func (s *lineOpStack) pop() *lineOp {
	if len(s.ops) <= 0 {
		return nil
	}

	op := s.ops[len(s.ops)-1]
	s.ops = s.ops[:len(s.ops)-1]
	return op
}

func (s *lineOpStack) clear() {
	s.ops = s.ops[:0]
}
