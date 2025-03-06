package buffer

import (
	"errors"
	"io"
)

var _ TextSource = (*PieceTableReader)(nil)

// PieceTableReader implements a [TextSource].
type PieceTableReader struct {
	*PieceTable
	// Index of the slice saves the continuous line number starting from zero.
	// The value contains the rune length of the line.
	lines      []lineInfo
	seekCursor int64
}

// ReadAt implements [io.ReaderAt].
func (r *PieceTableReader) ReadAt(p []byte, offset int64) (total int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if offset >= int64(r.seqBytes) {
		return 0, io.EOF
	}

	var expected = len(p)
	var bytes int64
	for n := r.pieces.Head(); n != r.pieces.tail; n = n.next {
		bytes += int64(n.byteLength)

		if bytes > offset {
			fragment := r.getBuf(n.source).getTextByRange(
				n.byteOff+n.byteLength-int(bytes-offset), // calculate the offset in the source buffer.
				int(bytes-offset))

			n := copy(p, fragment)
			p = p[n:]
			total += n
			offset += int64(n)

			if total >= expected {
				break
			}
		}

	}

	if total < expected {
		err = io.EOF
	}

	return
}

// Seek implements [io.Seeker].
func (r *PieceTableReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.seekCursor = offset
	case io.SeekCurrent:
		r.seekCursor += offset
	case io.SeekEnd:
		r.seekCursor = int64(r.seqBytes) + offset
	}
	return r.seekCursor, nil
}

// Read implements [io.Reader].
func (r *PieceTableReader) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.seekCursor)
	r.seekCursor += int64(n)
	return n, err
}

func (r *PieceTableReader) Text(buf []byte) []byte {
	if cap(buf) < int(r.seqBytes) {
		buf = make([]byte, r.seqBytes)
	}
	buf = buf[:r.seqBytes]
	r.Seek(0, io.SeekStart)
	n, _ := io.ReadFull(r, buf)
	buf = buf[:n]
	return buf
}

func (r *PieceTableReader) Lines() int {
	r.lines = r.lines[:0]
	for n := r.PieceTable.pieces.Head(); n != r.PieceTable.pieces.tail; n = n.next {
		pieceText := r.PieceTable.getBuf(n.source).getTextByRange(n.byteOff, n.byteLength)
		lines := r.parseLine(pieceText)
		if len(lines) > 0 {
			if len(r.lines) > 0 {
				lastLine := r.lines[len(r.lines)-1]
				if !lastLine.hasLineBreak {
					// merge with lastLine
					lines[0].length += lastLine.length
					r.lines = r.lines[:len(r.lines)-1]
				}
			}

			r.lines = append(r.lines, lines...)
		}
	}

	return len(r.lines)
}

// RuneOffset returns the byte offset for the rune at position runeOff.
func (r *PieceTableReader) RuneOffset(runeOff int) int {
	if r.seqLength == 0 {
		return 0
	}

	if runeOff >= r.seqLength {
		return r.seqBytes
	}

	n, off, byteOff := r.findPiece(runeOff)
	if n == nil {
		return r.seqBytes
	}

	return byteOff + r.getBuf(n.source).bytesForRange(n.offset, off)
}

func (r *PieceTableReader) ReadRuneAt(runeOff int) (rune, error) {
	n, off, _ := r.findPiece(runeOff)
	if n == nil {
		return 0, io.EOF
	}

	runes := r.getBuf(n.source).getTextByRuneRange(n.offset+off, 1)
	if len(runes) == 1 {
		return runes[0], nil
	}

	return 0, errors.New("read rune error")
}

// findPiece finds the starting piece of text that has runeOff in the range.
// It returns the found piece pointer, the rune offset relative to the start
// of the piece, and the bytes offset of the piece in the text sequence.
func (r *PieceTableReader) findPiece(runeOff int) (_ *piece, offset int, bytes int) {
	if runeOff >= r.seqLength {
		return nil, 0, 0
	}

	var runes int
	var n = r.pieces.Head()
	for {
		if n == r.pieces.tail {
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

func (r *PieceTableReader) parseLine(text []byte) []lineInfo {
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

func NewTextSource() *PieceTableReader {
	return &PieceTableReader{
		PieceTable: NewPieceTable([]byte("")),
	}
}
