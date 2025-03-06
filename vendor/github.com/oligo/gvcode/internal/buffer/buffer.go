package buffer

import (
	"bytes"
	"unicode/utf8"
)

// textBuffer is a byte buffer with a sparsing rune index to retrieve
// rune location to byte location efficiently.
type textBuffer struct {
	buf []byte
	runeOffIndex
	// Length of the  buffer in runes.
	length int
}

func newTextBuffer() *textBuffer {
	tb := &textBuffer{}
	tb.runeOffIndex = runeOffIndex{src: tb}

	return tb
}

// ReadRuneAt implements [runeReader].
func (tb *textBuffer) ReadRuneAt(byteOff int64) (rune, int, error) {
	b := make([]byte, utf8.UTFMax)
	reader := bytes.NewReader(tb.buf)
	n, err := reader.ReadAt(b, byteOff)
	b = b[:n]
	c, s := utf8.DecodeRune(b)
	return c, s, err
}

// set the inital buffer.
func (tb *textBuffer) set(buf []byte) int {
	tb.buf = buf
	tb.length = utf8.RuneCount(buf)
	return tb.length
}

// append to the buffer, returns the append rune offset, byte offset and rune length.
func (tb *textBuffer) append(buf []byte) (runeOff int, byteOff int, runeLen int) {
	byteOff = len(tb.buf)
	runeOff = tb.length

	tb.buf = append(tb.buf, buf...)
	runeLen = utf8.RuneCount(buf)
	tb.length += runeLen
	return
}

func (tb *textBuffer) bytesForRange(runeIdx int, runeLen int) int {
	start := tb.RuneOffset(runeIdx)
	end := tb.RuneOffset(runeIdx + runeLen)

	return end - start
}

func (tb *textBuffer) getTextByRange(byteIdx int, size int) []byte {
	if byteIdx < 0 || byteIdx >= len(tb.buf) {
		return nil
	}

	return tb.buf[byteIdx : byteIdx+size]
}

// getTextByRuneRange reads runes starting at the given rune offset.
func (tb *textBuffer) getTextByRuneRange(runeIdx int, size int) []rune {
	start := tb.RuneOffset(runeIdx)
	end := tb.RuneOffset(runeIdx + size)

	textBytes := tb.getTextByRange(start, end-start)
	runes := make([]rune, 0)
	for {
		c, s := utf8.DecodeRune(textBytes)
		if s == 0 {
			break
		}

		runes = append(runes, c)
		textBytes = textBytes[s:]
	}

	return runes
}
