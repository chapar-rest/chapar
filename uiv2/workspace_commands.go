package uiv2

import (
	"fmt"

	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

// tabsScope is the command palette scope that lists the open tabs.
const tabsScope = "tabs"

// Activate makes tab i the active tab.
func (w *Workspace) Activate(i int) {
	if i >= 0 && i < len(w.docs) {
		w.active = i
	}
}

// Step moves the active tab by delta, wrapping around the ends.
func (w *Workspace) Step(delta int) {
	n := len(w.docs)
	if n == 0 {
		return
	}
	w.active = ((w.active+delta)%n + n) % n
}

// commands returns the tab commands for the palette and their shortcuts.
// Enabled is false while the tab strip is not on screen, so the shortcuts do
// not act on tabs the user cannot see; Go to Tab stays available and calls
// reveal to bring the strip back first.
func (w *Workspace) commands(c *ui.Ctx, enabled bool, reveal func()) []*ui.Command {
	n := len(w.docs)
	on := enabled && n > 0
	cmds := []*ui.Command{
		ui.Section("Tabs"),
		ui.Cmd("tab.goto").Title("Go to Tab…").Shortcut("⌘P").Icon(icons.Search).Enable(n > 0).Run(func() {
			c.Commands().ShowScope(tabsScope, "Go to tab…")
		}),
		ui.Cmd("tab.close").Title("Close Tab").Shortcut("⌘W").Icon(icons.X).Enable(on).Run(func() {
			w.requestClose(w.active)
		}),
		ui.Cmd("tab.closeOthers").Title("Close Other Tabs").Enable(on && n > 1).Run(func() {
			active := w.active
			w.closeWhere(func(j int) bool { return j != active })
		}),
		ui.Cmd("tab.closeAll").Title("Close All Tabs").Enable(on).Run(func() {
			w.closeWhere(func(int) bool { return true })
		}),
		ui.Cmd("tab.next").Title("Next Tab").Shortcut("⌘⇧]").Icon(icons.ChevronRight).Enable(on && n > 1).Run(func() { w.Step(1) }),
		ui.Cmd("tab.prev").Title("Previous Tab").Shortcut("⌘⇧[").Icon(icons.ChevronLeft).Enable(on && n > 1).Run(func() { w.Step(-1) }),
	}
	// ⌘1–⌘8 jump to that tab and ⌘9 to the last, as in browsers.
	for k := 1; k <= 9; k++ {
		k := k
		cmds = append(cmds, ui.Cmd(fmt.Sprintf("tab.jump.%d", k)).
			Shortcut(fmt.Sprintf("⌘%d", k)).
			Hide(true).
			Enable(on && (k == 9 || k <= n)).
			Run(func() {
				if k == 9 {
					w.Activate(len(w.docs) - 1)
				} else {
					w.Activate(k - 1)
				}
			}))
	}

	cmds = append(cmds, ui.Section("Open tabs").Scope(tabsScope))
	for i, d := range w.docs {
		i := i
		detail := kindLabel(d.Kind())
		if d.Dirty() {
			detail += " · unsaved"
		}
		if i == w.active {
			detail += " · current"
		}
		cmds = append(cmds, ui.Item(fmt.Sprintf("tab.open.%d", i)).
			Title(d.Title()).
			Detail(detail).
			Icon(kindIcon(d.Kind())).
			Scope(tabsScope).
			Run(func() {
				reveal()
				w.Activate(i)
			}))
	}
	return cmds
}

func kindLabel(k container.Kind) string {
	switch k {
	case container.KindHTTP:
		return "HTTP request"
	case container.KindGRPC:
		return "gRPC request"
	case container.KindGraphQL:
		return "GraphQL request"
	case container.KindCollection:
		return "Collection"
	case container.KindEnv:
		return "Environment"
	}
	return string(k)
}

func kindIcon(k container.Kind) icons.Icon {
	switch k {
	case container.KindCollection:
		return icons.Folder
	case container.KindEnv:
		return icons.FolderPlus
	}
	return icons.File
}
