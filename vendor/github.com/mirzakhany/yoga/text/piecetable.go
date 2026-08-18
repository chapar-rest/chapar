// Package text provides the editor's text storage. It implements a piece table
// rather than a single growable string or a slice-of-lines, so that edits to a
// huge file are O(edit) instead of O(file): the original buffer is never
// mutated, inserts append to a separate "add" buffer, and the document is
// described purely as an ordered list of (buffer, start, length) spans.
package text

import "bytes"

// source identifies which backing buffer a piece reads from.
type source uint8

const (
	srcOriginal source = iota
	srcAdd
)

// piece is a contiguous span into one of the two immutable buffers.
type piece struct {
	src    source
	start  int
	length int
}

// PieceTable is an append-only, copy-free text buffer.
//
// Invariants:
//   - original and add are never mutated after the bytes they hold are written
//     (add only ever grows by appending), so existing pieces stay valid.
//   - the document content is the concatenation of pieces, in order.
type PieceTable struct {
	original []byte
	add      []byte
	pieces   []piece

	// cached flattened view + line offsets, invalidated on every edit.
	cacheValid bool
	flat       []byte
	lineStarts []int // byte offset of the first character of each line
}

// New creates a piece table seeded with initial content.
func New(initial []byte) *PieceTable {
	pt := &PieceTable{
		original: append([]byte(nil), initial...),
	}
	if len(pt.original) > 0 {
		pt.pieces = append(pt.pieces, piece{src: srcOriginal, start: 0, length: len(pt.original)})
	}
	return pt
}

func (pt *PieceTable) bufFor(s source) []byte {
	if s == srcAdd {
		return pt.add
	}
	return pt.original
}

// Len returns the document length in bytes.
func (pt *PieceTable) Len() int {
	n := 0
	for _, p := range pt.pieces {
		n += p.length
	}
	return n
}

// Bytes returns the full document as a freshly assembled byte slice. The result
// is cached until the next edit; callers must not mutate it.
func (pt *PieceTable) Bytes() []byte {
	pt.rebuildCache()
	return pt.flat
}

func (pt *PieceTable) rebuildCache() {
	if pt.cacheValid {
		return
	}
	pt.flat = pt.flat[:0]
	for _, p := range pt.pieces {
		buf := pt.bufFor(p.src)
		pt.flat = append(pt.flat, buf[p.start:p.start+p.length]...)
	}
	// Recompute line starts: line 0 starts at byte 0, each '\n' begins a new one.
	pt.lineStarts = pt.lineStarts[:0]
	pt.lineStarts = append(pt.lineStarts, 0)
	for i, b := range pt.flat {
		if b == '\n' {
			pt.lineStarts = append(pt.lineStarts, i+1)
		}
	}
	pt.cacheValid = true
}

// LineCount returns the number of lines (always >= 1).
func (pt *PieceTable) LineCount() int {
	pt.rebuildCache()
	return len(pt.lineStarts)
}

// MaxLineByteLen returns the longest line length in bytes (excluding newlines).
func (pt *PieceTable) MaxLineByteLen() int {
	_, l := pt.LongestLine()
	return l
}

// LongestLine returns the index and byte length (excluding newlines) of the
// longest line. Useful for estimating content width without shaping every line.
func (pt *PieceTable) LongestLine() (idx, byteLen int) {
	pt.rebuildCache()
	for i := 0; i < len(pt.lineStarts); i++ {
		start := pt.lineStarts[i]
		end := len(pt.flat)
		if i+1 < len(pt.lineStarts) {
			end = pt.lineStarts[i+1] - 1
		}
		if l := end - start; l > byteLen {
			byteLen = l
			idx = i
		}
	}
	return idx, byteLen
}

// Line returns the text of line i (0-based) without its trailing newline.
func (pt *PieceTable) Line(i int) string {
	pt.rebuildCache()
	if i < 0 || i >= len(pt.lineStarts) {
		return ""
	}
	start := pt.lineStarts[i]
	end := len(pt.flat)
	if i+1 < len(pt.lineStarts) {
		end = pt.lineStarts[i+1] - 1 // drop the '\n'
	}
	if end < start {
		end = start
	}
	return string(pt.flat[start:end])
}

// LineStart returns the byte offset where line i begins, enabling callers to map
// a line/column back to an absolute byte offset (used to align syntax tokens).
func (pt *PieceTable) LineStart(i int) int {
	pt.rebuildCache()
	if i < 0 {
		return 0
	}
	if i >= len(pt.lineStarts) {
		return len(pt.flat)
	}
	return pt.lineStarts[i]
}

// pieceAt locates the piece index and intra-piece offset for a document offset.
func (pt *PieceTable) pieceAt(offset int) (idx, within int) {
	acc := 0
	for i, p := range pt.pieces {
		if offset <= acc+p.length {
			return i, offset - acc
		}
		acc += p.length
	}
	return len(pt.pieces), 0
}

// Insert inserts s at the given byte offset.
func (pt *PieceTable) Insert(offset int, s []byte) {
	if len(s) == 0 {
		return
	}
	addStart := len(pt.add)
	pt.add = append(pt.add, s...)
	newPiece := piece{src: srcAdd, start: addStart, length: len(s)}

	idx, within := pt.pieceAt(offset)
	pt.cacheValid = false

	if idx >= len(pt.pieces) { // append at end
		pt.pieces = append(pt.pieces, newPiece)
		return
	}
	if within == 0 { // insert before piece idx
		pt.pieces = insertPieces(pt.pieces, idx, newPiece)
		return
	}
	// Split piece idx into [left | new | right].
	old := pt.pieces[idx]
	left := piece{src: old.src, start: old.start, length: within}
	right := piece{src: old.src, start: old.start + within, length: old.length - within}
	pt.pieces = replacePiece(pt.pieces, idx, left, newPiece, right)
}

// Delete removes length bytes starting at offset.
func (pt *PieceTable) Delete(offset, length int) {
	if length <= 0 {
		return
	}
	pt.cacheValid = false

	end := offset + length
	out := pt.pieces[:0:0] // build a fresh slice to avoid aliasing surprises
	out = make([]piece, 0, len(pt.pieces)+2)
	acc := 0
	for _, p := range pt.pieces {
		pStart, pEnd := acc, acc+p.length
		acc = pEnd
		if pEnd <= offset || pStart >= end { // fully outside the deletion
			out = append(out, p)
			continue
		}
		// Keep the portion before the deletion.
		if pStart < offset {
			out = append(out, piece{src: p.src, start: p.start, length: offset - pStart})
		}
		// Keep the portion after the deletion.
		if pEnd > end {
			keep := pEnd - end
			out = append(out, piece{src: p.src, start: p.start + (p.length - keep), length: keep})
		}
	}
	pt.pieces = out
}

func insertPieces(s []piece, idx int, p piece) []piece {
	s = append(s, piece{})
	copy(s[idx+1:], s[idx:])
	s[idx] = p
	return s
}

func replacePiece(s []piece, idx int, repl ...piece) []piece {
	out := make([]piece, 0, len(s)+len(repl)-1)
	out = append(out, s[:idx]...)
	for _, p := range repl {
		if p.length > 0 {
			out = append(out, p)
		}
	}
	out = append(out, s[idx+1:]...)
	return out
}

// Equal reports whether the document equals b (test/utility helper).
func (pt *PieceTable) Equal(b []byte) bool {
	return bytes.Equal(pt.Bytes(), b)
}
