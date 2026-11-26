// Package editor is the code editor widget for Yoga: a virtualized,
// piece-table text editor with Tree-sitter highlighting, language servers,
// find and replace, folding, multiple cursors and a minimap.
//
// Build an Editor once, in the app's Build, and place it with ui.ViewOf:
//
//	ed := editor.NewEditorFor("main.go", src)
//	...
//	ui.ViewOf(ed).Grow(1)
//
// Every shortcut is a row of DefaultEditorKeymap; see docs/editor-shortcuts.md.
package editor
