package buffer

// The specific rule that resolves ambiguity when an edit happens
// exactly at a marker's position.
type MarkerBias uint8

const (
	// BiasForward sets the rule that:
	//
	// - when the marker is at the start of the edit, marker moves
	// to the end of the inserted text.
	//
	// - when the marker is at the end of the edit, it is pushed to
	// the end of the inserted text
	BiasForward = iota

	// BiasBackward sets the rule that:
	//
	// - when the marker is at the start of the edit, it keeps staying
	// at the begining.
	//
	// - when the marker is at the end of the edit, it gets pulled to
	// the start of the inserted text.
	BiasBackward
)

// Marker is a text buffer annotation that tracks a logical location
// in the buffer over time. It tries to remain logically stationary
// even when the content changes.
type Marker struct {
	// The piece reference that the marker belongs to.
	piece *piece
	// rune offset of the marker in the piece relative to the piece offset.
	pieceOffset int
	// rune offset of the marker in the document.
	offset int
	bias   MarkerBias
}

func (m *Marker) update(p *piece, pieceOffset int) {
	m.piece = p
	m.pieceOffset = pieceOffset
}

// Offset returns the rune offset of the marker in the document.
func (m *Marker) Offset() int {
	return m.offset
}

func newMarker(p *piece, pieceOffset int, bais MarkerBias) *Marker {
	return &Marker{
		piece:       p,
		pieceOffset: pieceOffset,
		offset:      -1,
		bias:        bais,
	}
}
