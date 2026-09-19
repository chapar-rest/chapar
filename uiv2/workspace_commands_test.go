package uiv2

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/ui"
)

// pressTabKey registers the workspace commands in a fresh frame and sends one
// key through the command shortcuts, as the app does each frame.
func pressTabKey(t *testing.T, w *Workspace, visible bool, ev input.KeyEvent) *ui.Ctx {
	t.Helper()
	c := ui.New(nil, ui.NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	c.Commands().Register(w.commands(c, visible, func() {})...)
	c.Commands().Layout(c)
	kb := &input.Keyboard{Keys: []input.KeyEvent{ev}}
	c.Commands().Dispatch(kb)
	return c
}

func key(k input.Key, mods input.Mod) input.KeyEvent { return input.KeyEvent{Key: k, Mods: mods} }

func TestTabShortcutsNextPrevWrap(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c")
	w.active = 2
	pressTabKey(t, w, true, key(input.KeyRightBracket, input.ModSuper|input.ModShift))
	if w.active != 0 {
		t.Fatalf("next from the last tab should wrap to 0, got %d", w.active)
	}
	pressTabKey(t, w, true, key(input.KeyLeftBracket, input.ModSuper|input.ModShift))
	if w.active != 2 {
		t.Fatalf("previous from the first tab should wrap to 2, got %d", w.active)
	}
}

func TestTabShortcutsJump(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c", "d")
	pressTabKey(t, w, true, key(input.Key0+2, input.ModSuper))
	if w.active != 1 {
		t.Fatalf("⌘2: got %d want 1", w.active)
	}
	pressTabKey(t, w, true, key(input.Key0+9, input.ModSuper))
	if w.active != 3 {
		t.Fatalf("⌘9 should go to the last tab, got %d", w.active)
	}
	pressTabKey(t, w, true, key(input.Key0+6, input.ModSuper))
	if w.active != 3 {
		t.Fatalf("⌘6 with four tabs should do nothing, got %d", w.active)
	}
}

func TestTabShortcutCloseActive(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c")
	w.active = 1
	pressTabKey(t, w, true, key(input.KeyW, input.ModSuper))
	if got := docIDs(w); got != "ac" {
		t.Fatalf("⌘W: docs %q", got)
	}
}

func TestTabShortcutsOffWhenStripHidden(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c")
	pressTabKey(t, w, false, key(input.KeyW, input.ModSuper))
	pressTabKey(t, w, false, key(input.Key0+3, input.ModSuper))
	if got := docIDs(w); got != "abc" || w.active != 0 {
		t.Fatalf("shortcuts acted on a hidden strip: docs %q active %d", got, w.active)
	}
}

func TestGoToTabOpensScopedPalette(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b")
	c := pressTabKey(t, w, false, key(input.KeyP, input.ModSuper))
	if !c.Commands().Open {
		t.Fatal("⌘P should open the Go to Tab list even when the strip is hidden")
	}
}
