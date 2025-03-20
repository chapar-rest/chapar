package gvcode

import (
	"maps"
)

type bracketHandler struct {
	*textView
	idx bracketIdx
}

type bracket struct {
	r   rune
	pos int // rune offset.
}

type bracketIdx struct {
	idx []bracket
}

func (s *bracketIdx) push(item rune, pos int) {
	s.idx = append(s.idx, bracket{r: item, pos: pos})
}

func (s *bracketIdx) pop() (rune, int) {
	if len(s.idx) == 0 {
		return 0, 0
	}

	last := s.idx[len(s.idx)-1]
	s.idx = s.idx[:len(s.idx)-1]
	return last.r, last.pos
}

func (s *bracketIdx) peek() (rune, int) {
	if len(s.idx) == 0 {
		return 0, 0
	}

	last := s.idx[len(s.idx)-1]
	return last.r, last.pos
}

func (s *bracketIdx) depth() int {
	return len(s.idx)
}

func (s *bracketIdx) reset() {
	s.idx = s.idx[:0]
}

func (h *bracketHandler) rervesedBracketPairs() map[rune]rune {
	dest := make(map[rune]rune)
	for k, v := range h.BracketPairs {
		dest[v] = k
	}

	return dest
}

func (h *bracketHandler) checkBracket(r rune) (_ bool, isLeft bool) {
	if _, ok := h.BracketPairs[r]; ok {
		return true, true
	}

	if _, ok := h.rervesedBracketPairs()[r]; ok {
		return true, false
	}

	return false, false
}

func mergeMaps(sources ...map[rune]rune) map[rune]rune {
	dest := make(map[rune]rune)
	for _, src := range sources {
		maps.Copy(dest, src)
	}

	return dest
}

// NearestMatchingBrackets finds the nearest matching brackets of the caret.
func (e *bracketHandler) NearestMatchingBrackets() (left int, right int) {
	left, right = -1, -1
	start, end := e.Selection()
	if start != end {
		return
	}
	e.idx.reset()

	start = min(start, e.Len())
	nearest, err := e.src.ReadRuneAt(start)
	isBracket, _ := e.checkBracket(nearest)
	if err != nil || !isBracket {
		start = max(0, start-1)
		nearest, _ = e.src.ReadRuneAt(start)
	}

	if isBracket, isLeft := e.checkBracket(nearest); isBracket {
		if isLeft {
			left = start
		} else {
			right = start
		}
		e.idx.push(nearest, start)
	}

	rtlBrackets := e.rervesedBracketPairs()
	offset := start

	// find the left half.
	if left < 0 {
		for {
			offset = max(offset-1, 0)
			next, err := e.src.ReadRuneAt(offset)
			if err != nil {
				break
			}

			if br, ok := e.BracketPairs[next]; ok {
				if r, _ := e.idx.peek(); r == br {
					e.idx.pop()
					if right >= 0 && e.idx.depth() == 0 {
						left = offset
						break
					}
				} else {
					e.idx.push(next, offset)
					left = offset
					break
				}
			}

			// found a right half bracket.
			if _, ok := rtlBrackets[next]; ok {
				e.idx.push(next, offset)
			}

			if offset <= 0 {
				break
			}
		}
	}

	// find the right half.
	if right < 0 {
		for {
			offset = min(offset+1, e.Len())
			next, err := e.src.ReadRuneAt(offset)
			if err != nil {
				break
			}

			// found left half bracket
			if _, ok := e.BracketPairs[next]; ok {
				e.idx.push(next, offset)
			}

			// found a right half bracket.
			if bl, ok := rtlBrackets[next]; ok {
				if r, _ := e.idx.peek(); r == bl {
					e.idx.pop()
					if e.idx.depth() == 0 {
						right = offset
						break
					}
				} else {
					// Found a un-balanced bracket, drop it.
					//e.idx.push(next, offset)
				}
			}

			if offset >= e.Len() {
				break
			}

		}
	}

	return left, right
}
