package buffer

import (
	"sort"
)

// Entry for runeOffIndex. runes and bytes are cumulated runes and bytes count at
// a particular rune index.
type offsetEntry struct {
	runes int
	bytes int
}

// runeReader defines a ReadRuneAt API to reads the rune starting at the given byte offset, if any.
type runeReader interface {
	ReadRuneAt(byteOff int64) (rune, int, error)
}

// runeOffIndex adopts a sparse indexing algorithm to build a rune offset to byte offset index.
// To reduce the memory overhead of indexing every rune, we index every 50 runes.
//
// A runeReader has to be provided to build the index.
// The idea is brought from the textView of Gio's editor.
type runeOffIndex struct {
	src      runeReader
	offIndex []offsetEntry
}

// indexOfRune returns the latest rune index and byte offset no later than runeIndex.
func (r *runeOffIndex) indexOfRune(runeIndex int) offsetEntry {
	// Initialize index.
	if len(r.offIndex) == 0 {
		r.offIndex = append(r.offIndex, offsetEntry{})
	}

	i := sort.Search(len(r.offIndex), func(i int) bool {
		entry := r.offIndex[i]
		return entry.runes >= runeIndex
	})

	// Return the entry guaranteed to be less than or equal to runeIndex.
	if i > 0 {
		i--
	}

	return r.offIndex[i]
}

// runeOffset returns the byte offset of the source buffer's runeIndex'th rune.
// runeIndex must be a valid rune index.
func (r *runeOffIndex) RuneOffset(runeIndex int) int {
	const runesPerIndexEntry = 50
	entry := r.indexOfRune(runeIndex)
	lastEntry := r.offIndex[len(r.offIndex)-1].runes

	for entry.runes < runeIndex {
		if entry.runes > lastEntry && entry.runes%runesPerIndexEntry == runesPerIndexEntry-1 {
			r.offIndex = append(r.offIndex, entry)
		}
		_, s, _ := r.src.ReadRuneAt(int64(entry.bytes))
		entry.bytes += s
		entry.runes++
	}

	return entry.bytes
}
