package layout

// Host implements the "view is a function of state" pattern (SwiftUI's `body`,
// React's render) on top of the existing Scene contract. Instead of building the
// element tree once and surgically mutating `.Children` as state changes, an app
// gives Host a build function and calls Invalidate() whenever state changes;
// Host rebuilds the tree from scratch on the next frame and re-solves layout.
//
// Persistent state (component structs like a TextField or the data model) lives
// in the app and is *referenced* by build — only the lightweight Element wiring
// is rebuilt, so rebuilding is cheap and components keep their internal state.
//
//	h := layout.NewHost(func() *layout.Element {
//	    return layout.VStack(8, title.El, app.row(), app.list.El).Grow(1)
//	})
//	// in an event handler:
//	app.done = true
//	h.Invalidate()   // tree reflects the new state next frame
//
// A Scene delegates Root/Layout to the Host:
//
//	func (a *App) Root() *layout.Element  { return a.host.Root() }
//	func (a *App) Layout(w, h float32)    { a.host.Layout(w, h) }
type Host struct {
	build func() *Element
	root  *Element
	dirty bool
	vw, vh float32
}

// NewHost wraps a build function. The first Root/Layout call builds the tree.
func NewHost(build func() *Element) *Host {
	return &Host{build: build, dirty: true}
}

// Invalidate marks the tree stale so the next Root/Layout rebuilds it from
// current state. Cheap to call repeatedly within a frame; rebuild happens once.
func (h *Host) Invalidate() { h.dirty = true }

// Root returns the current tree, rebuilding and re-solving it first if state has
// been invalidated. Safe to call during Update (e.g. before Dispatch).
func (h *Host) Root() *Element {
	if h.dirty || h.root == nil {
		h.root = h.build()
		h.dirty = false
		if h.vw > 0 && h.vh > 0 {
			h.root.Calculate(h.vw, h.vh)
		}
	}
	return h.root
}

// Layout records the viewport and solves layout for the current tree. Called by
// the runtime each frame; a resize re-solves without needing an Invalidate.
func (h *Host) Layout(vw, vh float32) {
	h.vw, h.vh = vw, vh
	h.Root().Calculate(vw, vh)
}
