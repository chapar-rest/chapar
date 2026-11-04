package ui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/mirzakhany/yoga/input"
)

// The editor's keymap. Every editor shortcut is a row in a table of
// EditorBindings, so an app can list the shortcuts (EditorKeymapTable) and
// change them (SetKeymap, EditorDefaults.Keymap). Keys the table does not
// bind, such as typing, Enter, Backspace and the bare arrows, stay built in.

// EditorAction names something the editor can do from a shortcut.
type EditorAction string

// Editor actions with a default binding.
const (
	ActionSelectAll            EditorAction = "editor.selectAll"
	ActionCopy                 EditorAction = "editor.copy"
	ActionCut                  EditorAction = "editor.cut"
	ActionPaste                EditorAction = "editor.paste"
	ActionUndo                 EditorAction = "editor.undo"
	ActionRedo                 EditorAction = "editor.redo"
	ActionTriggerCompletion    EditorAction = "editor.triggerCompletion"
	ActionToggleSoftWrap       EditorAction = "editor.toggleSoftWrap"
	ActionAddNextOccurrence    EditorAction = "editor.addNextOccurrence"
	ActionSelectAllOccurrences EditorAction = "editor.selectAllOccurrences"
	ActionAddCursorAbove       EditorAction = "editor.addCursorAbove"
	ActionAddCursorBelow       EditorAction = "editor.addCursorBelow"

	ActionCursorWordLeft        EditorAction = "cursor.wordLeft"
	ActionCursorWordLeftSelect  EditorAction = "cursor.wordLeftSelect"
	ActionCursorWordRight       EditorAction = "cursor.wordRight"
	ActionCursorWordRightSelect EditorAction = "cursor.wordRightSelect"
	ActionCursorLineStart       EditorAction = "cursor.lineStart"
	ActionCursorLineStartSelect EditorAction = "cursor.lineStartSelect"
	ActionCursorLineEnd         EditorAction = "cursor.lineEnd"
	ActionCursorLineEndSelect   EditorAction = "cursor.lineEndSelect"
	ActionCursorFileStart       EditorAction = "cursor.fileStart"
	ActionCursorFileStartSelect EditorAction = "cursor.fileStartSelect"
	ActionCursorFileEnd         EditorAction = "cursor.fileEnd"
	ActionCursorFileEndSelect   EditorAction = "cursor.fileEndSelect"
	ActionDeleteWordLeft        EditorAction = "editor.deleteWordLeft"
	ActionDeleteWordRight       EditorAction = "editor.deleteWordRight"
	ActionDeleteLineLeft        EditorAction = "editor.deleteToLineStart"

	ActionDuplicateLinesDown EditorAction = "editor.duplicateLinesDown"
	ActionDuplicateLinesUp   EditorAction = "editor.duplicateLinesUp"
	ActionMoveLinesDown      EditorAction = "editor.moveLinesDown"
	ActionMoveLinesUp        EditorAction = "editor.moveLinesUp"
	ActionDeleteLines        EditorAction = "editor.deleteLines"
	ActionToggleComment      EditorAction = "editor.toggleComment"

	ActionFold      EditorAction = "editor.fold"
	ActionUnfold    EditorAction = "editor.unfold"
	ActionFoldAll   EditorAction = "editor.foldAll"
	ActionUnfoldAll EditorAction = "editor.unfoldAll"

	ActionFind               EditorAction = "editor.find"
	ActionReplace            EditorAction = "editor.replace"
	ActionFindNext           EditorAction = "find.next"
	ActionFindPrev           EditorAction = "find.previous"
	ActionFindClose          EditorAction = "find.close"
	ActionReplaceOne         EditorAction = "find.replace"
	ActionReplaceAll         EditorAction = "find.replaceAll"
	ActionFindToggleCase     EditorAction = "find.toggleCase"
	ActionFindToggleWord     EditorAction = "find.toggleWholeWord"
	ActionFindToggleRegex    EditorAction = "find.toggleRegex"
	ActionFindToggleInSel    EditorAction = "find.toggleInSelection"
	ActionFindTogglePreserve EditorAction = "find.togglePreserveCase"
	ActionFindToggleReplace  EditorAction = "find.toggleReplace"
)

// EditorScope is where a binding applies.
type EditorScope uint8

const (
	// ScopeText is the text area (the default).
	ScopeText EditorScope = iota
	// ScopeFind is either field of the find bar.
	ScopeFind
	// ScopeFindQuery is the find field only.
	ScopeFindQuery
	// ScopeFindReplace is the replace field only.
	ScopeFindReplace
)

// EditorBinding binds an action to a chord on macOS and on other platforms.
// Chords are written like "Mod+Alt+F": Mod is ⌘ on macOS and Ctrl elsewhere
// and accepts either key, as the editor always has; Cmd and Ctrl name that
// exact key. An empty chord leaves the action unbound on that platform. An
// action may have several bindings.
type EditorBinding struct {
	Action EditorAction
	Mac    string
	Other  string
	When   EditorScope
}

// DefaultEditorKeymap returns the built-in bindings. The slice is new on each
// call, so an app can edit it and pass it to SetKeymap.
func DefaultEditorKeymap() []EditorBinding {
	return []EditorBinding{
		{ActionSelectAll, "Mod+A", "Mod+A", ScopeText},
		{ActionCopy, "Mod+C", "Mod+C", ScopeText},
		{ActionCut, "Mod+X", "Mod+X", ScopeText},
		{ActionPaste, "Mod+V", "Mod+V", ScopeText},
		{ActionUndo, "Mod+Z", "Mod+Z", ScopeText},
		{ActionRedo, "Mod+Shift+Z", "Mod+Shift+Z", ScopeText},
		{ActionRedo, "Mod+Y", "Mod+Y", ScopeText},
		{ActionTriggerCompletion, "Mod+Space", "Mod+Space", ScopeText},
		{ActionToggleSoftWrap, "Mod+Alt+W", "Mod+Alt+W", ScopeText},
		{ActionAddNextOccurrence, "Mod+D", "Mod+D", ScopeText},
		{ActionSelectAllOccurrences, "Mod+Shift+L", "Mod+Shift+L", ScopeText},
		{ActionAddCursorAbove, "Mod+Alt+Up", "Ctrl+Alt+Up", ScopeText},
		{ActionAddCursorBelow, "Mod+Alt+Down", "Ctrl+Alt+Down", ScopeText},

		{ActionCursorWordLeft, "Alt+Left", "Ctrl+Left", ScopeText},
		{ActionCursorWordLeftSelect, "Alt+Shift+Left", "Ctrl+Shift+Left", ScopeText},
		{ActionCursorWordRight, "Alt+Right", "Ctrl+Right", ScopeText},
		{ActionCursorWordRightSelect, "Alt+Shift+Right", "Ctrl+Shift+Right", ScopeText},
		{ActionCursorLineStart, "Mod+Left", "Home", ScopeText},
		{ActionCursorLineStart, "Home", "", ScopeText},
		{ActionCursorLineStartSelect, "Mod+Shift+Left", "Shift+Home", ScopeText},
		{ActionCursorLineStartSelect, "Shift+Home", "", ScopeText},
		{ActionCursorLineEnd, "Mod+Right", "End", ScopeText},
		{ActionCursorLineEnd, "End", "", ScopeText},
		{ActionCursorLineEndSelect, "Mod+Shift+Right", "Shift+End", ScopeText},
		{ActionCursorLineEndSelect, "Shift+End", "", ScopeText},
		{ActionCursorFileStart, "Mod+Up", "Ctrl+Home", ScopeText},
		{ActionCursorFileStartSelect, "Mod+Shift+Up", "Ctrl+Shift+Home", ScopeText},
		{ActionCursorFileEnd, "Mod+Down", "Ctrl+End", ScopeText},
		{ActionCursorFileEndSelect, "Mod+Shift+Down", "Ctrl+Shift+End", ScopeText},
		{ActionDeleteWordLeft, "Alt+Backspace", "Ctrl+Backspace", ScopeText},
		{ActionDeleteWordRight, "Alt+Delete", "Ctrl+Delete", ScopeText},
		{ActionDeleteLineLeft, "Mod+Backspace", "", ScopeText},

		{ActionDuplicateLinesDown, "Alt+Shift+Down", "Alt+Shift+Down", ScopeText},
		{ActionDuplicateLinesUp, "Alt+Shift+Up", "Alt+Shift+Up", ScopeText},
		{ActionMoveLinesDown, "Alt+Down", "Alt+Down", ScopeText},
		{ActionMoveLinesUp, "Alt+Up", "Alt+Up", ScopeText},
		{ActionDeleteLines, "Mod+Shift+K", "Ctrl+Shift+K", ScopeText},
		{ActionToggleComment, "Mod+/", "Ctrl+/", ScopeText},
		{ActionFold, "Mod+Alt+[", "Ctrl+Shift+[", ScopeText},
		{ActionUnfold, "Mod+Alt+]", "Ctrl+Shift+]", ScopeText},
		{ActionFoldAll, "Mod+K Mod+0", "Ctrl+K Ctrl+0", ScopeText},
		{ActionUnfoldAll, "Mod+K Mod+J", "Ctrl+K Ctrl+J", ScopeText},

		{ActionFind, "Mod+F", "Mod+F", ScopeText},
		{ActionReplace, "Mod+Alt+F", "Ctrl+H", ScopeText},
		{ActionFind, "Mod+F", "Mod+F", ScopeFind},
		{ActionReplace, "Mod+Alt+F", "Ctrl+H", ScopeFind},
		{ActionFindNext, "Enter", "Enter", ScopeFindQuery},
		{ActionFindPrev, "Shift+Enter", "Shift+Enter", ScopeFindQuery},
		{ActionFindNext, "Mod+G", "F3", ScopeFind},
		{ActionFindPrev, "Mod+Shift+G", "Shift+F3", ScopeFind},
		{ActionReplaceOne, "Enter", "Enter", ScopeFindReplace},
		{ActionReplaceAll, "Mod+Enter", "Ctrl+Enter", ScopeFind},
		{ActionFindClose, "Escape", "Escape", ScopeFind},
		{ActionFindToggleCase, "Mod+Alt+C", "Alt+C", ScopeFind},
		{ActionFindToggleWord, "Mod+Alt+W", "Alt+W", ScopeFind},
		{ActionFindToggleRegex, "Mod+Alt+R", "Alt+R", ScopeFind},
		{ActionFindToggleInSel, "Mod+Alt+L", "Alt+L", ScopeFind},
		{ActionFindTogglePreserve, "Mod+Alt+P", "Alt+P", ScopeFind},
	}
}

// editorActionTitles names actions for shortcut listings.
var editorActionTitles = map[EditorAction]string{
	ActionSelectAll:             "Select All",
	ActionCopy:                  "Copy",
	ActionCut:                   "Cut",
	ActionPaste:                 "Paste",
	ActionUndo:                  "Undo",
	ActionRedo:                  "Redo",
	ActionTriggerCompletion:     "Trigger Completion",
	ActionToggleSoftWrap:        "Toggle Soft Wrap",
	ActionAddNextOccurrence:     "Add Next Occurrence",
	ActionSelectAllOccurrences:  "Select All Occurrences",
	ActionAddCursorAbove:        "Add Cursor Above",
	ActionAddCursorBelow:        "Add Cursor Below",
	ActionCursorWordLeft:        "Cursor Word Left",
	ActionCursorWordLeftSelect:  "Select Word Left",
	ActionCursorWordRight:       "Cursor Word Right",
	ActionCursorWordRightSelect: "Select Word Right",
	ActionCursorLineStart:       "Cursor Line Start",
	ActionCursorLineStartSelect: "Select to Line Start",
	ActionCursorLineEnd:         "Cursor Line End",
	ActionCursorLineEndSelect:   "Select to Line End",
	ActionCursorFileStart:       "Go to Start of File",
	ActionCursorFileStartSelect: "Select to Start of File",
	ActionCursorFileEnd:         "Go to End of File",
	ActionCursorFileEndSelect:   "Select to End of File",
	ActionDeleteWordLeft:        "Delete Word Left",
	ActionDeleteWordRight:       "Delete Word Right",
	ActionDeleteLineLeft:        "Delete to Line Start",
	ActionDuplicateLinesDown:    "Duplicate Lines Down",
	ActionDuplicateLinesUp:      "Duplicate Lines Up",
	ActionMoveLinesDown:         "Move Lines Down",
	ActionMoveLinesUp:           "Move Lines Up",
	ActionDeleteLines:           "Delete Lines",
	ActionToggleComment:         "Toggle Line Comment",
	ActionFold:                  "Fold",
	ActionUnfold:                "Unfold",
	ActionFoldAll:               "Fold All",
	ActionUnfoldAll:             "Unfold All",
	ActionFind:                  "Find",
	ActionReplace:               "Replace",
	ActionFindNext:              "Find Next",
	ActionFindPrev:              "Find Previous",
	ActionFindClose:             "Close Find",
	ActionReplaceOne:            "Replace",
	ActionReplaceAll:            "Replace All",
	ActionFindToggleCase:        "Toggle Match Case",
	ActionFindToggleWord:        "Toggle Whole Word",
	ActionFindToggleRegex:       "Toggle Regular Expression",
	ActionFindToggleInSel:       "Toggle Find in Selection",
	ActionFindTogglePreserve:    "Toggle Preserve Case",
	ActionFindToggleReplace:     "Toggle Replace",
}

// editorActions runs each action. A false return means the action did not
// apply, and the key goes on to the editor's built-in handling.
var editorActions = map[EditorAction]func(e *Editor) bool{}

func init() {
	for act, fn := range map[EditorAction]func(e *Editor) bool{
		ActionSelectAll: func(e *Editor) bool { e.SelectAll(); return true },
		ActionCopy:      func(e *Editor) bool { e.copyAll(frameClipboard()); return true },
		ActionCut:       func(e *Editor) bool { e.cutAll(frameClipboard()); return true },
		ActionPaste:     func(e *Editor) bool { e.pasteAll(frameClipboard()); return true },
		ActionUndo:      func(e *Editor) bool { e.extra = nil; e.Undo(); return true },
		ActionRedo:      func(e *Editor) bool { e.extra = nil; e.Redo(); return true },
		ActionTriggerCompletion: func(e *Editor) bool {
			e.requestCompletion()
			return true
		},
		ActionToggleSoftWrap:       func(e *Editor) bool { e.SetSoftWrap(!e.SoftWrap); return true },
		ActionAddNextOccurrence:    func(e *Editor) bool { e.AddNextOccurrence(); return true },
		ActionSelectAllOccurrences: func(e *Editor) bool { e.SelectAllOccurrences(); return true },
		ActionAddCursorAbove:       func(e *Editor) bool { e.AddCursorLine(-1); return true },
		ActionAddCursorBelow:       func(e *Editor) bool { e.AddCursorLine(1); return true },

		ActionFind:               func(e *Editor) bool { e.openSearch(false); return true },
		ActionReplace:            func(e *Editor) bool { e.openSearch(true); return true },
		ActionFindNext:           func(e *Editor) bool { e.searchNext(); return true },
		ActionFindPrev:           func(e *Editor) bool { e.searchPrev(); return true },
		ActionFindClose:          func(e *Editor) bool { e.closeSearchToMatch(); return true },
		ActionReplaceOne:         func(e *Editor) bool { e.doReplace(); return true },
		ActionReplaceAll:         func(e *Editor) bool { e.doReplaceAll(); return true },
		ActionFindToggleCase:     func(e *Editor) bool { e.toggleFind(&e.search.caseSens); return true },
		ActionFindToggleWord:     func(e *Editor) bool { e.toggleFind(&e.search.wholeWord); return true },
		ActionFindToggleRegex:    func(e *Editor) bool { e.toggleFind(&e.search.regex); return true },
		ActionFindToggleInSel:    func(e *Editor) bool { e.toggleInSelection(); return true },
		ActionFindTogglePreserve: func(e *Editor) bool { e.search.preserveCase = !e.search.preserveCase; return true },
		ActionFindToggleReplace:  func(e *Editor) bool { e.toggleReplaceRow(); return true },
	} {
		editorActions[act] = fn
	}
}

// Run performs an editor action by name, as its shortcut would, and reports
// whether the action exists. Apps bind their own commands to it.
func (e *Editor) Run(action EditorAction) bool {
	fn := editorActions[action]
	if fn == nil {
		return false
	}
	fn(e)
	return true
}

// Keymap returns the bindings this editor uses.
func (e *Editor) Keymap() []EditorBinding { return e.keymapSource() }

// SetKeymap replaces this editor's bindings; nil goes back to
// EditorDefaults.Keymap, or the built-in DefaultEditorKeymap.
func (e *Editor) SetKeymap(b []EditorBinding) {
	e.keymapOwn = b
	e.keymapC = nil
}

func (e *Editor) keymapSource() []EditorBinding {
	switch {
	case e.keymapOwn != nil:
		return e.keymapOwn
	case EditorDefaults.Keymap != nil:
		return EditorDefaults.Keymap
	}
	return defaultKeymap()
}

var defaultKeymapCache []EditorBinding

func defaultKeymap() []EditorBinding {
	if defaultKeymapCache == nil {
		defaultKeymapCache = DefaultEditorKeymap()
	}
	return defaultKeymapCache
}

// editorMac selects the macOS column of the keymap; tests flip it.
var editorMac = runtime.GOOS == "darwin"

// editorChord is a chord with exact modifiers, except that primary accepts
// either ⌘ or Ctrl.
type editorChord struct {
	key     input.Key
	mods    input.Mod // Shift, Alt, and literal Ctrl/Super
	primary bool
}

func (c editorChord) matches(ev input.KeyEvent) bool {
	if ev.Key != c.key {
		return false
	}
	const sa = input.ModShift | input.ModAlt
	if ev.Mods&sa != c.mods&sa {
		return false
	}
	if c.primary {
		return ev.Mods.Primary()
	}
	const cs = input.ModCtrl | input.ModSuper
	return ev.Mods&cs == c.mods&cs
}

// parseEditorChord parses a keymap chord such as "Mod+Shift+L".
func parseEditorChord(s string) (editorChord, error) {
	parts := splitChord(strings.TrimSpace(s))
	if len(parts) == 0 {
		return editorChord{}, fmt.Errorf("empty chord")
	}
	var c editorChord
	for _, p := range parts[:len(parts)-1] {
		switch normalizeModToken(p) {
		case "mod":
			c.primary = true
		case "shift":
			c.mods |= input.ModShift
		case "alt", "option", "opt":
			c.mods |= input.ModAlt
		case "ctrl", "control":
			c.mods |= input.ModCtrl
		case "cmd", "command", "super", "meta", "⌘":
			c.mods |= input.ModSuper
		default:
			return editorChord{}, fmt.Errorf("unknown modifier %q", p)
		}
	}
	k, ok := parseKeyToken(parts[len(parts)-1])
	if !ok {
		return editorChord{}, fmt.Errorf("unknown key %q", parts[len(parts)-1])
	}
	c.key = k
	return c, nil
}

// label renders a chord for menus and tooltips: "⌘⌥F" on macOS, "Ctrl+Alt+F"
// elsewhere.
func (c editorChord) label(mac bool) string {
	key := keyLabel(c.key)
	if mac {
		var b strings.Builder
		if c.mods.Has(input.ModCtrl) {
			b.WriteRune('⌃')
		}
		if c.mods.Has(input.ModAlt) {
			b.WriteRune('⌥')
		}
		if c.mods.Has(input.ModShift) {
			b.WriteRune('⇧')
		}
		if c.primary || c.mods.Has(input.ModSuper) {
			b.WriteRune('⌘')
		}
		b.WriteString(key)
		return b.String()
	}
	switch c.key {
	case input.KeyEnter:
		key = "Enter"
	case input.KeyLeft:
		key = "Left"
	case input.KeyRight:
		key = "Right"
	case input.KeyUp:
		key = "Up"
	case input.KeyDown:
		key = "Down"
	}
	var parts []string
	if c.primary || c.mods.Has(input.ModCtrl) {
		parts = append(parts, "Ctrl")
	}
	if c.mods.Has(input.ModSuper) {
		parts = append(parts, "Win")
	}
	if c.mods.Has(input.ModAlt) {
		parts = append(parts, "Alt")
	}
	if c.mods.Has(input.ModShift) {
		parts = append(parts, "Shift")
	}
	return strings.Join(append(parts, key), "+")
}

type compiledBinding struct {
	action EditorAction
	chord  editorChord
	then   editorChord // the second chord of a two-chord sequence
	seq    bool
	when   EditorScope
}

// parseEditorSeq parses one chord, or two separated by a space ("Mod+K
// Mod+0").
func parseEditorSeq(s string) (first, then editorChord, seq bool, err error) {
	parts := seqParts(s)
	switch len(parts) {
	case 1:
		first, err = parseEditorChord(parts[0])
		return first, then, false, err
	case 2:
		if first, err = parseEditorChord(parts[0]); err != nil {
			return
		}
		then, err = parseEditorChord(parts[1])
		return first, then, true, err
	}
	return first, then, false, fmt.Errorf("bad shortcut %q", s)
}

func (b compiledBinding) label(mac bool) string {
	if b.seq {
		return b.chord.label(mac) + " " + b.then.label(mac)
	}
	return b.chord.label(mac)
}

// compiledKeymap is the current platform's column of a keymap, parsed.
type compiledKeymap struct {
	src  []EditorBinding
	mac  bool
	rows []compiledBinding
}

func compileKeymap(src []EditorBinding, mac bool) *compiledKeymap {
	km := &compiledKeymap{src: src, mac: mac}
	for _, b := range src {
		s := b.Other
		if mac {
			s = b.Mac
		}
		if s == "" {
			continue
		}
		c, then, seq, err := parseEditorSeq(s)
		if err != nil {
			continue // a bad user binding is skipped, not fatal
		}
		km.rows = append(km.rows, compiledBinding{action: b.Action, chord: c, then: then, seq: seq, when: b.When})
	}
	return km
}

func (e *Editor) keymap() *compiledKeymap {
	src := e.keymapSource()
	if km := e.keymapC; km != nil && km.mac == editorMac && sameBindings(km.src, src) {
		return km
	}
	e.keymapC = compileKeymap(src, editorMac)
	return e.keymapC
}

func sameBindings(a, b []EditorBinding) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// scopeMatches reports whether a binding for when applies at scope at.
func scopeMatches(when, at EditorScope) bool {
	if when == at {
		return true
	}
	return when == ScopeFind && (at == ScopeFindQuery || at == ScopeFindReplace)
}

// lookup finds the action bound to ev at scope at.
func (km *compiledKeymap) lookup(ev input.KeyEvent, at EditorScope) (EditorAction, bool) {
	for _, r := range km.rows {
		if !r.seq && scopeMatches(r.when, at) && r.chord.matches(ev) {
			return r.action, true
		}
	}
	return "", false
}

// runKey runs the action ev is bound to at scope at, and reports whether it
// took the key. The first chord of a two-chord sequence waits for the next
// key, which then belongs to the sequence whether or not it completes one.
func (e *Editor) runKey(ev input.KeyEvent, at EditorScope) bool {
	km := e.keymap()
	if p := e.keyPending; p != nil {
		e.keyPending = nil
		for _, r := range km.rows {
			if r.seq && r.chord == *p && scopeMatches(r.when, at) && r.then.matches(ev) {
				if fn := editorActions[r.action]; fn != nil {
					fn(e)
				}
				break
			}
		}
		return true
	}
	for _, r := range km.rows {
		if r.seq && scopeMatches(r.when, at) && r.chord.matches(ev) {
			c := r.chord
			e.keyPending = &c
			return true
		}
	}
	act, ok := km.lookup(ev, at)
	if !ok {
		return false
	}
	fn := editorActions[act]
	return fn != nil && fn(e)
}

// KeyPending reports whether the editor is waiting for the second chord of
// a two-chord shortcut.
func (e *Editor) KeyPending() bool { return e.keyPending != nil }

// ShortcutLabel is the label of the first chord bound to action on this
// platform, such as "⌘F", or "" when it is unbound.
func (e *Editor) ShortcutLabel(action EditorAction) string {
	for _, r := range e.keymap().rows {
		if r.action == action {
			return r.label(editorMac)
		}
	}
	return ""
}

// EditorShortcut is one row of a shortcut listing.
type EditorShortcut struct {
	Action EditorAction
	Title  string
	Mac    string // label, such as "⌘⌥F"
	Other  string // label, such as "Ctrl+H"
	When   EditorScope
}

// EditorKeymapTable lists bindings with display labels for both platforms,
// in keymap order, for shortcut references and settings pages.
func EditorKeymapTable(bindings []EditorBinding) []EditorShortcut {
	if bindings == nil {
		bindings = DefaultEditorKeymap()
	}
	out := make([]EditorShortcut, 0, len(bindings))
	lbl := func(s string, mac bool) string {
		if s == "" {
			return ""
		}
		c, then, seq, err := parseEditorSeq(s)
		if err != nil {
			return s
		}
		return compiledBinding{chord: c, then: then, seq: seq}.label(mac)
	}
	for _, b := range bindings {
		out = append(out, EditorShortcut{
			Action: b.Action,
			Title:  editorActionTitles[b.Action],
			Mac:    lbl(b.Mac, true),
			Other:  lbl(b.Other, false),
			When:   b.When,
		})
	}
	return out
}
