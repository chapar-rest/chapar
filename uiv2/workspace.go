package uiv2

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/mirzakhany/yoga/ui"
)

// Workspace is the unified tab strip for requests, collections, and environments.
type Workspace struct {
	tabs    []ui.TabModel
	docs    []container.Container
	active  int
	deps    func() container.Deps
	confirm *confirm
	onTrees func()
}

type confirm struct {
	open    bool
	message string
	onYes   func()
}

func newWorkspace(deps func() container.Deps, confirmHost *confirm) *Workspace {
	return &Workspace{deps: deps, confirm: confirmHost}
}

func (w *Workspace) OpenRequest(req *domain.Request) {
	w.open(container.OpenSpec{Request: req, Deps: w.containerDeps(req.MetaData.ID)})
}

func (w *Workspace) OpenCollection(col *domain.Collection) {
	w.open(container.OpenSpec{Collection: col, Deps: w.containerDeps(col.MetaData.ID)})
}

func (w *Workspace) OpenEnv(env *domain.Environment) {
	w.open(container.OpenSpec{Env: env, Deps: w.containerDeps(env.MetaData.ID)})
}

func (w *Workspace) containerDeps(id string) container.Deps {
	d := w.deps()
	d.Report = container.Reporter{
		Dirty: func(dirty bool) { w.setDirty(id, dirty) },
		Title: func(title string) { w.setTitle(id, title) },
		Error: d.Report.Error,
		Saved: func() {
			if d.Catalog != nil {
				_ = d.Catalog.Load()
			}
			if w.onTrees != nil {
				w.onTrees()
			}
		},
	}
	return d
}

func (w *Workspace) open(spec container.OpenSpec) {
	ct, err := openContainer(spec)
	if err != nil {
		w.deps().ShowError(err)
		return
	}
	for i, existing := range w.docs {
		if existing.ID() == ct.ID() {
			ct.Close()
			w.active = i
			return
		}
	}
	w.docs = append(w.docs, ct)
	w.tabs = append(w.tabs, ui.TabModel{Title: ct.Title(), Modified: ct.Dirty()})
	w.active = len(w.docs) - 1
}

func (w *Workspace) setDirty(id string, dirty bool) {
	for i, d := range w.docs {
		if d.ID() == id {
			w.tabs[i].Modified = dirty
			return
		}
	}
}

func (w *Workspace) setTitle(id, title string) {
	for i, d := range w.docs {
		if d.ID() == id {
			w.tabs[i].Title = title
			return
		}
	}
}

func (w *Workspace) Active() container.Container {
	if w.active < 0 || w.active >= len(w.docs) {
		return nil
	}
	return w.docs[w.active]
}

func (w *Workspace) CloseAll() {
	for _, d := range w.docs {
		d.Close()
	}
	w.docs = nil
	w.tabs = nil
	w.active = 0
}

func (w *Workspace) requestClose(i int) {
	if i < 0 || i >= len(w.docs) {
		return
	}
	if w.docs[i].Dirty() && w.confirm != nil {
		idx := i
		w.confirm.open = true
		w.confirm.message = fmt.Sprintf("%q has unsaved changes. Close anyway?", w.docs[i].Title())
		w.confirm.onYes = func() { w.drop(idx) }
		return
	}
	w.drop(i)
}

func (w *Workspace) drop(i int) {
	if i < 0 || i >= len(w.docs) {
		return
	}
	w.docs[i].Close()
	w.docs = append(w.docs[:i], w.docs[i+1:]...)
	w.tabs = append(w.tabs[:i], w.tabs[i+1:]...)
	if w.active >= len(w.docs) {
		w.active = len(w.docs) - 1
	}
	if w.active < 0 {
		w.active = 0
	}
}

func (w *Workspace) HasDirty() bool {
	for _, d := range w.docs {
		if d.Dirty() {
			return true
		}
	}
	return false
}

func (w *Workspace) SaveActive() {
	if a := w.Active(); a != nil {
		if err := a.Save(); err != nil {
			w.deps().ShowError(err)
		}
	}
}

func (w *Workspace) SendActive() {
	if a := w.Active(); a != nil {
		a.Send()
	}
}

func (w *Workspace) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	if len(w.docs) == 0 {
		return ui.Center(ui.Muted("Open a request or environment from the tree")).Grow(1)
	}
	if w.active < 0 || w.active >= len(w.docs) {
		w.active = 0
	}
	for i, d := range w.docs {
		w.tabs[i].Modified = d.Dirty()
		w.tabs[i].Title = d.Title()
	}
	body := w.docs[w.active].Layout(c)
	return ui.Column(
		ui.Tabs("workspace-tabs", w.tabs).
			Selected(w.active).
			OnSelectItem(func(i int, _ string) { w.active = i }).
			OnTabClose(func(i int) { w.requestClose(i) }).
			TabBackground(th.Background),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(body).Grow(1),
	).Grow(1)
}
