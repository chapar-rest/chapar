package buffer

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type bufSrc uint8
type action uint8

const (
	original bufSrc = iota
	modify
)

const (
	actionUnknown action = iota
	actionInsert
	actionErase
	actionReplace
)

var debugEnabled = false

// PieceTable implements a piece table data structure.
// See the following resources for more information:
//
// https://en.wikipedia.org/wiki/Piece_table
//
// https://www.cs.unm.edu/~crowley/papers/sds.pdf
//
// This implementation is heavily inspired by the design described in
// James Brown's Piece Chain(http://www.catch22.net/tuts/neatpad/piece-chains).
type PieceTable struct {
	originalBuf *textBuffer
	modifyBuf   *textBuffer
	// Length of the text sequence in runes.
	seqLength int
	// bytes size of the text sequence.
	seqBytes int

	// undo stack and redo stack
	undoStack *pieceRangeStack
	redoStack *pieceRangeStack
	// piece list
	pieces *pieceList

	// last action and action position in rune offset in the text sequence.
	lastAction       action
	lastActionEndIdx int
	// last inserted piece, for insertion optimization purpose.
	lastInsertPiece *piece
	// changed tracks whether the sequence content has changed since the last call to Changed.
	changed bool
	// setting a batchId to group
	currentBatch *int
}

func NewPieceTable(text []byte) *PieceTable {
	pt := &PieceTable{
		originalBuf: newTextBuffer(),
		modifyBuf:   newTextBuffer(),
		pieces:      newPieceList(),
		undoStack:   &pieceRangeStack{},
		redoStack:   &pieceRangeStack{},
	}
	pt.init(text)

	return pt
}

func (pt *PieceTable) SetText(text []byte) {
	pt.originalBuf = newTextBuffer()
	pt.modifyBuf = newTextBuffer()
	pt.pieces = newPieceList()
	pt.undoStack.clear()
	pt.redoStack.clear()
	pt.seqBytes = 0
	pt.seqLength = 0
	pt.lastAction = actionUnknown
	pt.lastActionEndIdx = 0
	pt.lastInsertPiece = nil
	pt.changed = false
	pt.currentBatch = nil

	pt.init(text)
}

// Initialize the piece table with the text by adding the text to the original buffer,
// and create the first piece point to the buffer.
func (pt *PieceTable) init(text []byte) {
	_, _, runeCnt := pt.addToBuffer(original, text)
	if runeCnt <= 0 {
		return
	}

	piece := &piece{
		source:     original,
		offset:     0,
		length:     runeCnt,
		byteOff:    0,
		byteLength: len(text),
	}

	pt.pieces.Append(piece)
	pt.seqLength = piece.length
	pt.seqBytes = piece.byteLength
}

func (pt *PieceTable) addToBuffer(source bufSrc, text []byte) (int, int, int) {
	if len(text) <= 0 {
		return 0, 0, 0
	}

	if source == original {
		return 0, 0, pt.originalBuf.set(text)
	}

	return pt.modifyBuf.append(text)
}

func (pt *PieceTable) getBuf(source bufSrc) *textBuffer {
	if source == original {
		return pt.originalBuf
	}

	return pt.modifyBuf
}

func (pt *PieceTable) recordAction(action action, runeIndex int) {
	pt.lastAction = action
	pt.lastActionEndIdx = runeIndex
}

func (pt *PieceTable) push2UndoStack(rng, newRng *pieceRange) {
	if pt.currentBatch != nil {
		rng.batchId = pt.currentBatch
	}

	pt.undoStack.push(rng)
	// swap link the new piece into the sequence
	rng.Swap(newRng)
}

// Insert insert text at the logical position specifed by runeIndex. runeIndex is measured by rune.
// There are 2 scenarios need to be handled:
//  1. Insert in the middle of a piece.
//  2. Insert at the boundary of two pieces.
func (pt *PieceTable) Insert(runeIndex int, text string) bool {
	if runeIndex > pt.seqLength || runeIndex < 0 || text == "" {
		return false
	}

	pt.redoStack.clear()

	// special-case: inserting at the end of a prior insertion at a piece boundary.
	if pt.tryAppendToLastPiece(runeIndex, text) {
		pt.changed = true
		return true
	}

	oldPiece, inRuneOff := pt.pieces.FindPiece(runeIndex)

	if inRuneOff == 0 {
		pt.insertAtBoundary(runeIndex, text, oldPiece)
	} else {
		pt.insertInMiddle(runeIndex, text, oldPiece, inRuneOff)
	}

	pt.changed = true
	return true
}

// Check if this insert action can be optimized by merging the input with previous one.
// multiple characters input won't be merged.
func (pt *PieceTable) tryAppendToLastPiece(runeIndex int, text string) bool {
	if pt.lastAction != actionInsert ||
		runeIndex != pt.lastActionEndIdx ||
		pt.lastInsertPiece == nil ||
		utf8.RuneCountInString(text) > 1 {
		return false
	}

	_, _, textRunes := pt.addToBuffer(modify, []byte(text))
	if textRunes <= 0 {
		return false
	}

	pt.lastInsertPiece.length += textRunes
	pt.lastInsertPiece.byteLength += len(text)

	pt.seqLength += textRunes
	pt.seqBytes += len(text)
	pt.recordAction(actionInsert, runeIndex+textRunes)

	return true
}

func (pt *PieceTable) insertAtBoundary(runeIndex int, text string, oldPiece *piece) {
	textRuneOff, textByteOff, textRunes := pt.addToBuffer(modify, []byte(text))

	newPiece := &piece{
		source:     modify,
		offset:     textRuneOff,
		length:     textRunes,
		byteOff:    textByteOff,
		byteLength: len(text),
	}
	pt.lastInsertPiece = newPiece

	// insertion is at the boundary of 2 pieces.
	oldPieces := &pieceRange{
		cursor: CursorPos{Start: runeIndex, End: runeIndex},
	}
	oldPieces.AsBoundary(oldPiece)

	newPieces := &pieceRange{}
	newPieces.Append(newPiece)
	// swap link the new piece into the sequence
	pt.push2UndoStack(oldPieces, newPieces)
	pt.seqLength += textRunes
	pt.seqBytes += len(text)
	pt.recordAction(actionInsert, runeIndex+textRunes)
}

func (pt *PieceTable) insertInMiddle(runeIndex int, text string, oldPiece *piece, inRuneOff int) {
	textRuneOff, textByteOff, textRunes := pt.addToBuffer(modify, []byte(text))

	newPiece := &piece{
		source:     modify,
		offset:     textRuneOff,
		length:     textRunes,
		byteOff:    textByteOff,
		byteLength: len(text),
	}
	pt.lastInsertPiece = newPiece

	// preserve the old pieces as a pieceRange, and push to the undo stack.
	oldPieces := &pieceRange{
		cursor: CursorPos{Start: runeIndex, End: runeIndex},
	}
	oldPieces.Append(oldPiece)

	// spilt the old piece into 2 new pieces, and insert the newly added text.
	newPieces := &pieceRange{}

	// Append the left part of the old piece.
	byteLen := pt.getBuf(oldPiece.source).bytesForRange(oldPiece.offset, inRuneOff)
	newPieces.Append(&piece{
		source:     oldPiece.source,
		offset:     oldPiece.offset,
		length:     inRuneOff,
		byteOff:    oldPiece.byteOff,
		byteLength: byteLen,
	})

	// Then the newly added piece.
	newPieces.Append(newPiece)

	//  And the right part of the old piece.
	byteOff := pt.getBuf(oldPiece.source).RuneOffset(oldPiece.offset + inRuneOff)
	byteLen = pt.getBuf(oldPiece.source).bytesForRange(oldPiece.offset+inRuneOff, oldPiece.length-inRuneOff)
	newPieces.Append(&piece{
		source:     oldPiece.source,
		offset:     oldPiece.offset + inRuneOff,
		length:     oldPiece.length - inRuneOff,
		byteOff:    byteOff,
		byteLength: byteLen,
	})

	pt.push2UndoStack(oldPieces, newPieces)
	pt.seqLength += textRunes
	pt.seqBytes += len(text)
	pt.recordAction(actionInsert, runeIndex+textRunes)
}

// undoRedo restores operation saved in src to dest. If there is a valid batchId, the src stack
// is searched for continuous batched operations to restore one by one.
// It returns all cursor postion(start and end rune offset) after restoration for all the operation.
func (pt *PieceTable) undoRedo(src *pieceRangeStack, dest *pieceRangeStack) ([]CursorPos, bool) {
	if src.depth() <= 0 {
		return nil, false
	}

	restoreFunc := func(rng *pieceRange) CursorPos {
		newRuneLen, newBytes := rng.Size()

		// restore to the old piece range.
		rng.Restore()
		// add the restored range onto the destination stack
		dest.push(rng)

		lastRuneLen, lastBytes := rng.Size()
		pt.seqLength += newRuneLen - lastRuneLen
		pt.seqBytes += newBytes - lastBytes
		pt.changed = true
		return rng.cursor
	}

	cursors := make([]CursorPos, 0)
	// remove the next event from the source stack
	rng := src.peek()
	batchId := rng.batchId
	if batchId == nil {
		src.pop()
		cursors = append(cursors, restoreFunc(rng))
		return cursors, true
	}

	for batchId != nil && rng != nil && batchId == rng.batchId {
		src.pop()
		cursors = append(cursors, restoreFunc(rng))

		// Try the next.
		rng = src.peek()
	}

	return cursors, true
}

func (pt *PieceTable) Erase(startOff, endOff int) bool {
	cursor := CursorPos{Start: startOff, End: endOff}

	if startOff > endOff {
		startOff, endOff = endOff, startOff
	}

	if endOff > pt.seqLength {
		endOff = pt.seqLength
	}

	if startOff == endOff {
		return false
	}

	pt.redoStack.clear()
	defer func() { pt.changed = true }()

	startPiece, inRuneOff := pt.pieces.FindPiece(startOff)

	oldPieces := &pieceRange{
		cursor: cursor,
	}
	oldPieces.Append(startPiece)

	newPieces := &pieceRange{}
	bytesErased := 0

	// start and end all in the middle of the startPiece. Keep both sides of the startPiece.
	if inRuneOff > 0 && endOff-startOff <= startPiece.length-inRuneOff {
		leftByteLen := pt.getBuf(startPiece.source).bytesForRange(startPiece.offset, inRuneOff)

		rightByteLen := pt.getBuf(startPiece.source).bytesForRange(startPiece.offset+inRuneOff+endOff-startOff, startPiece.length-inRuneOff-(endOff-startOff))
		rightByteOff := pt.getBuf(startPiece.source).RuneOffset(startPiece.offset + inRuneOff + endOff - startOff)
		newPieces.Append(&piece{
			source:     startPiece.source,
			offset:     startPiece.offset,
			length:     inRuneOff,
			byteOff:    startPiece.byteOff,
			byteLength: leftByteLen,
		})

		if rightByteLen > 0 {
			newPieces.Append(&piece{
				source:     startPiece.source,
				offset:     startPiece.offset + inRuneOff + endOff - startOff,
				length:     startPiece.length - inRuneOff - (endOff - startOff),
				byteOff:    rightByteOff,
				byteLength: rightByteLen,
			})
		}
		bytesErased += startPiece.byteLength - leftByteLen - rightByteLen
		pt.push2UndoStack(oldPieces, newPieces)
		pt.seqLength -= endOff - startOff
		pt.seqBytes -= bytesErased
		return true
	}

	// Delete start in the middle of a piece. Split the piece and keep the left part.
	if inRuneOff > 0 {
		leftByteLen := pt.getBuf(startPiece.source).bytesForRange(startPiece.offset, inRuneOff)

		newPieces.Append(&piece{
			source:     startPiece.source,
			offset:     startPiece.offset,
			length:     inRuneOff,
			byteOff:    startPiece.byteOff,
			byteLength: leftByteLen,
		})
		bytesErased += startPiece.byteLength - leftByteLen
		startPiece = startPiece.next
	}

	offset := startOff
	n := startPiece
	for ; n != pt.pieces.tail; n = n.next {
		if offset >= endOff {
			break
		}

		if offset < endOff && offset+n.length > endOff {
			// Found the last affected piece, and the delete stops in the middle of it.
			// Keep the right part of the end piece.
			byteLen := pt.getBuf(n.source).bytesForRange(n.offset+endOff-offset, n.length-(endOff-offset))
			byteOff := pt.getBuf(n.source).RuneOffset(n.offset + endOff - offset)

			newPieces.Append(&piece{
				source:     n.source,
				offset:     n.offset + endOff - offset,
				length:     n.length - (endOff - offset),
				byteOff:    byteOff,
				byteLength: byteLen,
			})
			bytesErased += n.byteLength - byteLen
		} else {
			bytesErased += n.byteLength
		}

		// push pieces in the middle and the end piece to undo stack.
		if n != startPiece {
			oldPieces.Append(n)
		}

		offset += n.length
	}

	if newPieces.Length() == 0 {
		newPieces.AsBoundary(n)
	}

	// swap link the new piece into the sequence
	pt.push2UndoStack(oldPieces, newPieces)
	pt.seqLength -= endOff - startOff
	pt.seqBytes -= bytesErased

	//pt.recordAction(actionInsert, runeIndex+textRunes)
	return true
}

// Replace removes text from startOff to endOff(exclusive), and insert text at the position of startOff.
func (pt *PieceTable) Replace(startOff, endOff int, text string) bool {
	defer pt.Inspect()

	if endOff > pt.seqLength {
		endOff = pt.seqLength
	}

	if startOff == endOff && text != "" {
		return pt.Insert(startOff, text)
	}

	if text == "" {
		return pt.Erase(startOff, endOff)
	}

	pt.GroupOp()
	defer pt.UnGroupOp()

	if !pt.Erase(startOff, endOff) {
		return false
	}

	return pt.Insert(startOff, text)
}

func (pt *PieceTable) Undo() ([]CursorPos, bool) {
	defer pt.Inspect()
	return pt.undoRedo(pt.undoStack, pt.redoStack)
}

func (pt *PieceTable) Redo() ([]CursorPos, bool) {
	defer pt.Inspect()
	return pt.undoRedo(pt.redoStack, pt.undoStack)
}

// Group operations such as insert, earase or replace in a batch.
// Nested call share the same single batch.
func (pt *PieceTable) GroupOp() {
	if pt.currentBatch == nil {
		pt.currentBatch = new(int)
	}

	*pt.currentBatch += 1
}

// Ungroup a batch. Latter insert, earase or replace operations outside of
// a group is not batched.
func (pt *PieceTable) UnGroupOp() {
	*pt.currentBatch--
	if *pt.currentBatch <= 0 {
		pt.currentBatch = nil
	}
}

// Size returns the total length of the document data in runes.
func (pt *PieceTable) Len() int {
	return pt.seqLength
}

func (pt *PieceTable) Changed() bool {
	c := pt.changed
	pt.changed = false
	return c
}

// Inspect prints the internal of the piece table. For debug purpose only.
func (pt *PieceTable) Inspect() {
	if !debugEnabled {
		return
	}

	fmt.Println("<---------------------------------------------------->")
	fmt.Println("\toriginal buffer size: ", pt.originalBuf.length)
	fmt.Println("\tmodify buffer size: ", pt.modifyBuf.length)
	fmt.Println("\ttext sequence size: ", pt.seqLength)
	fmt.Println("\tpieces: ", pt.pieces.Length())
	id := 0
	for n := pt.pieces.Head(); n != pt.pieces.tail; n = n.next {
		content := string(pt.getBuf(n.source).getTextByRange(n.byteOff, n.byteLength))
		content = strings.ReplaceAll(content, "\n", "\\n")
		content = strings.ReplaceAll(content, "\t", "\\t")
		fmt.Printf("\t\t#%d: source: %d, range:(%d:%d), text: %s\n", id, n.source, n.offset, n.offset+n.length, content)
		id++
	}
	fmt.Printf("\traw modify buffer: %s\n", string(pt.modifyBuf.buf))
	fmt.Println("<---------------------------------------------------->")
}

// Set debug mode or not. In debug mode, the internal states of PieceTable is printed to console.
func SetDebug(enable bool) {
	debugEnabled = enable
}
