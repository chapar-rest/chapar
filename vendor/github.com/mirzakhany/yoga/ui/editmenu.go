package ui

import (
	"github.com/mirzakhany/yoga/input"
)

// editMenuWidth fits the longest label plus its shortcut hint.
const editMenuWidth = 200

// textEditing is what the standard edit menu drives. TextInput and Editor
// both implement it.
type textEditing interface {
	CanUndo() bool
	CanRedo() bool
	HasSelection() bool
	Undo()
	Redo()
	Cut()
	Copy() bool
	Paste()
	SelectAll()
}

// editMenuFlags say which groups of the standard edit menu apply.
type editMenuFlags struct {
	editable bool // offer Undo, Redo, Cut and Paste
	copyable bool // offer Cut and Copy (false for a password field)
}

// editMenuItems builds the standard Undo / Redo / Cut / Copy / Paste / Select
// All menu for t, disabling what cannot apply right now.
func editMenuItems(t textEditing, f editMenuFlags) []MenuItem {
	canCopy := f.copyable && t.HasSelection()
	var items []MenuItem
	if f.editable {
		items = append(items,
			MenuItem{Label: "Undo", Shortcut: editShortcut(input.KeyZ, 0), Disabled: !t.CanUndo(), OnSelect: t.Undo},
			MenuItem{Label: "Redo", Shortcut: editShortcut(input.KeyZ, input.ModShift), Disabled: !t.CanRedo(), OnSelect: t.Redo},
			MenuSeparator,
			MenuItem{Label: "Cut", Shortcut: editShortcut(input.KeyX, 0), Disabled: !canCopy, OnSelect: t.Cut},
		)
	}
	items = append(items, MenuItem{Label: "Copy", Shortcut: editShortcut(input.KeyC, 0), Disabled: !canCopy, OnSelect: func() { t.Copy() }})
	if f.editable {
		clip := frameClipboard()
		canPaste := clip != nil && clip.Get() != ""
		items = append(items, MenuItem{Label: "Paste", Shortcut: editShortcut(input.KeyV, 0), Disabled: !canPaste, OnSelect: t.Paste})
	}
	return append(items,
		MenuSeparator,
		MenuItem{Label: "Select All", Shortcut: editShortcut(input.KeyA, 0), OnSelect: t.SelectAll},
	)
}

func editShortcut(k input.Key, mods input.Mod) string {
	return Chord{Key: k, Mods: mods, primary: true}.Label()
}

// editMenu is the right-click menu a text widget owns.
type editMenu struct {
	menu *Menu
}

// open shows items at (x, y) and reports whether there was anything to show.
func (em *editMenu) open(items []MenuItem, x, y float32) bool {
	if len(items) == 0 {
		return false
	}
	if em.menu == nil {
		em.menu = NewMenu(editMenuWidth, items)
	} else {
		em.menu.SetItems(items)
	}
	em.menu.OpenAt(x, y)
	if em.menu.markPaint != nil {
		em.menu.markPaint()
	}
	return true
}

func (em *editMenu) isOpen() bool { return em.menu != nil && em.menu.Open }

func (em *editMenu) close() {
	if em.menu != nil {
		em.menu.Close()
	}
}

// layout registers the open menu as an overlay for this frame. Escape closes
// it and is swallowed, so it does not also cancel the widget's edit or search.
// Call it from the owning widget's Layout.
func (em *editMenu) layout(c *Ctx) {
	if !em.isOpen() {
		return
	}
	if kb := c.Keyboard(); kb != nil {
		kept := kb.Keys[:0]
		for _, ev := range kb.Keys {
			if ev.Key == input.KeyEscape {
				em.close()
				c.MarkNeedsPaint()
				continue
			}
			kept = append(kept, ev)
		}
		kb.Keys = kept
		if !em.isOpen() {
			return
		}
	}
	em.menu.BindPaint(c)
	c.Overlay(em.menu.overlay())
}
