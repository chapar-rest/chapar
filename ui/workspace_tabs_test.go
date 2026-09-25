package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/ui/container"
)

func tabsWorkspace(confirm func(title, message string, onYes func()), ids ...string) *Workspace {
	w := newWorkspace(func() container.Deps { return container.Deps{} }, confirm)
	for _, id := range ids {
		d := &fakeDoc{id: id}
		w.docs = append(w.docs, d)
		w.tabs = append(w.tabs, ui.TabModel{Title: id})
	}
	return w
}

func docIDs(w *Workspace) string {
	s := ""
	for _, d := range w.docs {
		s += d.ID()
	}
	return s
}

func menuItem(t *testing.T, items []ui.MenuItem, label string) ui.MenuItem {
	t.Helper()
	for _, it := range items {
		if it.Label == label {
			return it
		}
	}
	t.Fatalf("no %q in tab menu", label)
	return ui.MenuItem{}
}

func TestTabMenuCloseOthers(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c", "d")
	w.active = 3
	menuItem(t, w.tabMenu(1), "Close Others").OnSelect()
	if got := docIDs(w); got != "b" {
		t.Fatalf("docs after Close Others: %q", got)
	}
	if w.Active().ID() != "b" {
		t.Fatalf("active: %q", w.Active().ID())
	}
}

func TestTabMenuCloseToTheRightAndLeft(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c", "d")
	w.active = 2
	menuItem(t, w.tabMenu(1), "Close to the Left").OnSelect()
	if got := docIDs(w); got != "bcd" {
		t.Fatalf("docs after Close to the Left: %q", got)
	}
	if w.Active().ID() != "c" {
		t.Fatalf("active should stay on c, got %q", w.Active().ID())
	}
	menuItem(t, w.tabMenu(0), "Close to the Right").OnSelect()
	if got := docIDs(w); got != "b" {
		t.Fatalf("docs after Close to the Right: %q", got)
	}
	items := w.tabMenu(0)
	if !menuItem(t, items, "Close Others").Disabled || !menuItem(t, items, "Close to the Right").Disabled ||
		!menuItem(t, items, "Close to the Left").Disabled {
		t.Fatal("a single tab should disable the Close Others/Right/Left items")
	}
}

func TestTabMenuCloseSavedKeepsDirty(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c")
	w.docs[1].(*fakeDoc).dirty = true
	menuItem(t, w.tabMenu(0), "Close Saved").OnSelect()
	if got := docIDs(w); got != "b" {
		t.Fatalf("docs after Close Saved: %q", got)
	}
}

func TestTabMenuCloseAllConfirmsOnceForDirty(t *testing.T) {
	var asks int
	var yes func()
	w := tabsWorkspace(func(_, _ string, onYes func()) { asks++; yes = onYes }, "a", "b", "c")
	w.docs[0].(*fakeDoc).dirty = true
	w.docs[2].(*fakeDoc).dirty = true
	menuItem(t, w.tabMenu(1), "Close All").OnSelect()
	if asks != 1 {
		t.Fatalf("confirm asked %d times, want 1", asks)
	}
	if len(w.docs) != 3 {
		t.Fatal("nothing should close before the user confirms")
	}
	yes()
	if len(w.docs) != 0 {
		t.Fatalf("docs after confirming Close All: %q", docIDs(w))
	}
}

func TestDropLeftOfActiveKeepsActiveDoc(t *testing.T) {
	w := tabsWorkspace(nil, "a", "b", "c")
	w.active = 2
	w.drop(0)
	if w.Active().ID() != "c" {
		t.Fatalf("closing a tab left of the active one moved the active tab to %q", w.Active().ID())
	}
}
