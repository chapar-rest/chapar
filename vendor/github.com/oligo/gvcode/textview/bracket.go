package textview

import (
	"maps"
)

// Built-in quote pairs used for auto-insertion. This can be
// replaced by applying the WithQuotePairs editor option.
var builtinQuotePairs = map[rune]rune{
	'\'': '\'',
	'"':  '"',
	'`':  '`',
}

// Built-in bracket pairs used for auto-insertion. This can be
// replaced by applying the WithBracketPairs editor option.
var builtinBracketPairs = map[rune]rune{
	'(': ')',
	'{': '}',
	'[': ']',
}

type runePairs struct {
	// opening maps opening rune to closing rune.
	opening map[rune]rune
	// closing  maps closing rune to opening rune.
	closing map[rune]rune
}

func (p *runePairs) set(pairs map[rune]rune) {
	if p.opening == nil {
		p.opening = make(map[rune]rune)
		p.closing = make(map[rune]rune)
	} else {
		clear(p.opening)
		clear(p.closing)
	}

	maps.Copy(p.opening, pairs)
	for k, v := range pairs {
		p.closing[v] = k
	}
}

// getClosing get the closing rune for r, if r is a opening rune.
func (p *runePairs) getClosing(r rune) (rune, bool) {
	r, exists := p.opening[r]
	return r, exists
}

// getOpening get the opening rune for r, if r is the closing rune.
func (p *runePairs) getOpening(r rune) (rune, bool) {
	r, ok := p.closing[r]
	return r, ok
}

// contains check if r is contained in p, it also reports whether r is
// a opening part.
func (p *runePairs) contains(r rune) (bool, bool) {
	if _, exists := p.getClosing(r); exists {
		return true, true
	}
	if _, exists := p.getOpening(r); exists {
		return true, false
	}

	return false, false
}

// bracketQuotes holds configured bracket pairs and quote pairs.
type bracketsQuotes struct {
	// A set of quote pairs that can be auto-completed when the opening half is entered.
	quotePairs *runePairs
	// A set of bracket pairs that can be auto-completed when the opening half is entered.
	bracketPairs *runePairs
}

// SetBrackets set bracket pairs using a opening bracket to closing bracket map.
func (bq *bracketsQuotes) SetBrackets(bracketPairs map[rune]rune) {
	if bq.bracketPairs == nil {
		bq.bracketPairs = &runePairs{}
	}
	bq.bracketPairs.set(bracketPairs)
}

// SetQuotes set quote pairs using a opening quote to closing quote map.
func (bq *bracketsQuotes) SetQuotes(quotePairs map[rune]rune) {
	if bq.quotePairs == nil {
		bq.quotePairs = &runePairs{}
	}

	bq.quotePairs.set(quotePairs)
}

// Contains check if r is contained in the configured quotes, and if r is a opening
// quote.
func (bq bracketsQuotes) ContainsQuote(r rune) (_ bool, isOpening bool) {
	if bq.quotePairs == nil {
		bq.SetQuotes(builtinQuotePairs)
	}

	return bq.quotePairs.contains(r)
}

// Contains check if r is contained in the configured brackets, and if r is a opening
// bracket.
func (bq bracketsQuotes) ContainsBracket(r rune) (_ bool, isOpening bool) {
	if bq.bracketPairs == nil {
		bq.SetBrackets(builtinBracketPairs)
	}

	return bq.bracketPairs.contains(r)
}

// GetCounterpart check if r is contained in the configured brackets or quotes, it returns the
// counterpart of r and whether r is a opening half.
func (bq *bracketsQuotes) GetCounterpart(r rune) (_ rune, isOpening bool) {
	if bq.quotePairs == nil {
		bq.SetQuotes(builtinQuotePairs)
	}
	if bq.bracketPairs == nil {
		bq.SetBrackets(builtinBracketPairs)
	}

	if r, exists := bq.bracketPairs.getClosing(r); exists {
		return r, true
	}
	if r, exists := bq.bracketPairs.getOpening(r); exists {
		return r, false
	}

	if r, exists := bq.quotePairs.getClosing(r); exists {
		return r, true
	}
	if r, exists := bq.quotePairs.getOpening(r); exists {
		return r, false
	}

	return rune(0), false
}

func (bq *bracketsQuotes) GetOpeningBracket(r rune) (rune, bool) {
	if bq.bracketPairs == nil {
		bq.SetBrackets(builtinBracketPairs)
	}

	return bq.bracketPairs.getOpening(r)
}

func (bq *bracketsQuotes) GetClosingBracket(r rune) (rune, bool) {
	if bq.bracketPairs == nil {
		bq.SetBrackets(builtinBracketPairs)
	}

	return bq.bracketPairs.getClosing(r)
}

func (bq *bracketsQuotes) GetOpeningQuote(r rune) (rune, bool) {
	if bq.quotePairs == nil {
		bq.SetQuotes(builtinQuotePairs)
	}
	return bq.quotePairs.getOpening(r)
}

func (bq *bracketsQuotes) GetClosingQuote(r rune) (rune, bool) {
	if bq.quotePairs == nil {
		bq.SetQuotes(builtinQuotePairs)
	}
	return bq.quotePairs.getClosing(r)
}

// NearestMatchingBrackets finds the nearest matching brackets of the caret.
func (e *TextView) NearestMatchingBrackets() (left int, right int) {
	left, right = -1, -1
	start, end := e.Selection()
	if start != end {
		return
	}

	stack := &bracketStack{}
	stack.reset()

	start = min(start, e.Len())
	nearest, err := e.src.ReadRuneAt(start)
	isBracket, _ := e.BracketsQuotes.ContainsBracket(nearest)
	if err != nil || !isBracket {
		start = max(0, start-1)
		nearest, _ = e.src.ReadRuneAt(start)
	}

	if isBracket, isLeft := e.BracketsQuotes.ContainsBracket(nearest); isBracket {
		if isLeft {
			left = start
		} else {
			right = start
		}
		stack.push(nearest, start)
	}

	offset := start

	// find the left half.
	if left < 0 {
		for {
			offset = max(offset-1, 0)
			next, err := e.src.ReadRuneAt(offset)
			if err != nil {
				break
			}

			// Check if next is a opening bracket.
			if br, ok := e.BracketsQuotes.GetClosingBracket(next); ok {
				if r, _ := stack.peek(); r == br {
					stack.pop()
					if right >= 0 && stack.depth() == 0 {
						left = offset
						break
					}
				} else {
					if r == 0 {
						stack.push(next, offset)
						left = offset
					} else {
						// Found unbalanced bracket, drop it.
					}
					break
				}
			}

			// found a right half bracket.
			if exists, isOpening := e.BracketsQuotes.ContainsBracket(next); exists && !isOpening {
				stack.push(next, offset)
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
			if _, isOpening := e.BracketsQuotes.ContainsBracket(next); isOpening {
				stack.push(next, offset)
			}

			// found a right half bracket.
			if bl, ok := e.BracketsQuotes.GetOpeningBracket(next); ok {
				if r, _ := stack.peek(); r == bl {
					stack.pop()
					if stack.depth() == 0 {
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

type bracketPos struct {
	r   rune
	pos int // rune offset.
}

type bracketStack struct {
	idx []bracketPos
}

func (s *bracketStack) push(item rune, pos int) {
	s.idx = append(s.idx, bracketPos{r: item, pos: pos})
}

func (s *bracketStack) pop() (rune, int) {
	if len(s.idx) == 0 {
		return 0, 0
	}

	last := s.idx[len(s.idx)-1]
	s.idx = s.idx[:len(s.idx)-1]
	return last.r, last.pos
}

func (s *bracketStack) peek() (rune, int) {
	if len(s.idx) == 0 {
		return 0, 0
	}

	last := s.idx[len(s.idx)-1]
	return last.r, last.pos
}

func (s *bracketStack) depth() int {
	return len(s.idx)
}

func (s *bracketStack) reset() {
	s.idx = s.idx[:0]
}
