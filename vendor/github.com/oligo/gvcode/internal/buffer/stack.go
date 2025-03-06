package buffer

// pieceRangeStack for undo & redo.
type pieceRangeStack struct {
	ranges []*pieceRange
}

func (s *pieceRangeStack) push(rng *pieceRange) {
	s.ranges = append(s.ranges, rng)
}

func (s *pieceRangeStack) peek() *pieceRange {
	if len(s.ranges) <= 0 {
		return nil
	}

	return s.ranges[len(s.ranges)-1]
}

func (s *pieceRangeStack) pop() *pieceRange {
	if len(s.ranges) <= 0 {
		return nil
	}

	rng := s.ranges[len(s.ranges)-1]
	s.ranges = s.ranges[:len(s.ranges)-1]
	return rng
}

func (s *pieceRangeStack) depth() int {
	return len(s.ranges)
}

func (s *pieceRangeStack) clear() {
	s.ranges = s.ranges[:0]
}
