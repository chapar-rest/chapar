package buffer

import "io"

// TextSource provides data for editor.
//
// Basic editing operations, such as insert, delete, replace,
// undo/redo are supported. If used with GroupOp and UnGroupOp,
// the undo and redo operations can be batched.
type TextSource interface {
	io.Seeker
	io.Reader
	io.ReaderAt

	// ReadRuneAt reads the rune starting at the given rune offset, if any.
	ReadRuneAt(runeOff int) (rune, error)

	// RuneOffset returns the byte offset for the rune at position runeIndex.
	RuneOffset(runeIndex int) int

	// Lines returns the total number of lines/paragraphs of the source.
	Lines() int

	//Text returns the contents of the editor.
	Text(buf []byte) []byte

	// Len is the length of the editor contents, in runes.
	Len() int

	// SetText reset the buffer and replace the content of the buffer with the provided text.
	SetText(text []byte)

	// Insert insert text at the logical position specifed by runeIndex measured by rune.
	Insert(runeIndex int, text string) bool
	// Delete text from startOff to endOff(exclusive).
	Erase(startOff, endOff int) bool
	// Replace replace text from startOff to endOff(exclusive) with text.
	Replace(startOff, endOff int, text string) bool

	// Undo the last insert, erase, or replace, or a group of operations.
	// It returns all the cursor positions after undo.
	Undo() ([]CursorPos, bool)
	// Redo the last insert, erase, or replace, or a group of operations.
	// It returns all the cursor positions after undo.
	Redo() ([]CursorPos, bool)

	// Group operations such as insert, earase or replace in a batch.
	// Nested call share the same single batch.
	GroupOp()

	// Ungroup a batch. Latter insert, earase or replace operations outside of
	// a group is not batched.
	UnGroupOp()

	// Changed report whether the contents have changed since the last call to Changed.
	Changed() bool
}
