package buffer

import (
	"errors"
	"io"
)

var _ TextSource = (*PieceTable)(nil)

// RuneOffset returns the byte offset for the rune at position runeOff.
func (pt *PieceTable) RuneOffset(runeOff int) int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.seqLength == 0 {
		return 0
	}

	if runeOff >= pt.seqLength {
		return pt.seqBytes
	}

	n, off, byteOff := pt.findPiece(runeOff)
	if n == nil {
		return pt.seqBytes
	}

	return byteOff + pt.getBuf(n.source).bytesForRange(n.offset, off)
}

func (pt *PieceTable) ReadRuneAt(runeOff int) (rune, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	n, off, _ := pt.findPiece(runeOff)
	if n == nil {
		return 0, io.EOF
	}

	runes := pt.getBuf(n.source).getTextByRuneRange(n.offset+off, 1)
	if len(runes) == 1 {
		return runes[0], nil
	}

	return 0, errors.New("read rune error")
}

// findPiece finds the starting piece of text that has runeOff in the range.
// It returns the found piece pointer, the rune offset relative to the start
// of the piece, and the bytes offset of the piece in the text sequence.
func (pt *PieceTable) findPiece(runeOff int) (_ *piece, offset int, bytes int) {
	if runeOff >= pt.seqLength {
		return nil, 0, 0
	}

	var runes int
	var n = pt.pieces.Head()
	for {
		if n == pt.pieces.tail {
			return nil, 0, bytes
		}

		if runes+n.length > runeOff {
			return n, runeOff - runes, bytes
		}

		runes += n.length
		bytes += n.byteLength
		n = n.next
	}
}

func (r *PieceTable) parseLine(text []byte) []lineInfo {
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

func (pt *PieceTable) Lines() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	pt.lines = pt.lines[:0]
	for n := pt.pieces.Head(); n != pt.pieces.tail; n = n.next {
		pieceText := pt.getBuf(n.source).getTextByRange(n.byteOff, n.byteLength)
		lines := pt.parseLine(pieceText)
		if len(lines) > 0 {
			if len(pt.lines) > 0 {
				lastLine := pt.lines[len(pt.lines)-1]
				if !lastLine.hasLineBreak {
					// merge with lastLine
					lines[0].length += lastLine.length
					pt.lines = pt.lines[:len(pt.lines)-1]
				}
			}

			pt.lines = append(pt.lines, lines...)
		}
	}

	return len(pt.lines)
}

// pieceTableReader implements a [TextSource].
type pieceTableReader struct {
	src        TextSource
	seekCursor int64
}

// Seek implements [io.Seeker].
func (r *pieceTableReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.seekCursor = offset
	case io.SeekCurrent:
		r.seekCursor += offset
	case io.SeekEnd:
		r.seekCursor = int64(r.src.Size()) + offset
	}
	return r.seekCursor, nil
}

// Read implements [io.Reader].
func (r *pieceTableReader) Read(p []byte) (int, error) {
	n, err := r.src.ReadAt(p, r.seekCursor)
	r.seekCursor += int64(n)
	return n, err
}

func (r *pieceTableReader) ReadAll(buf []byte) []byte {
	size := r.src.Size()
	if cap(buf) < size {
		buf = make([]byte, size)
	}
	buf = buf[:size]
	r.Seek(0, io.SeekStart)
	n, _ := io.ReadFull(r, buf)
	buf = buf[:n]
	return buf
}

func NewTextSource() *PieceTable {
	return NewPieceTable([]byte(""))
}

func NewReader(src TextSource) TextReader {
	return &pieceTableReader{src: src}
}
