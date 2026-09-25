package ui

import (
	"runtime"
	"strings"
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/container"
)

// fakeDoc stands in for a request tab holding a large response body.
type fakeDoc struct {
	id     string
	body   []byte
	closed bool
	dirty  bool
}

func (d *fakeDoc) ID() string               { return d.id }
func (d *fakeDoc) Kind() container.Kind     { return container.KindHTTP }
func (d *fakeDoc) Title() string            { return d.id }
func (d *fakeDoc) Dirty() bool              { return d.dirty }
func (d *fakeDoc) Layout(c *ui.Ctx) ui.View { return nil }
func (d *fakeDoc) Close()                   { d.closed = true }
func (d *fakeDoc) Save() error              { return nil }
func (d *fakeDoc) Send()                    {}

const fakeBodySize = 8 << 20

func newFakeDoc(id string) *fakeDoc {
	return &fakeDoc{id: id, body: make([]byte, fakeBodySize)}
}

func heapAllocMB() float64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapAlloc) / 1e6
}

// TestClosedTabIsReleased is the guard on tab close: once a tab is closed the
// workspace must not keep its container reachable, or the response body it
// holds stays live for as long as the workspace does.
func TestClosedTabIsReleased(t *testing.T) {
	w := newWorkspace(func() container.Deps { return container.Deps{} }, nil)

	const tabs = 4
	for i := 0; i < tabs; i++ {
		d := newFakeDoc(string(rune('a' + i)))
		w.docs = append(w.docs, d)
		w.tabs = append(w.tabs, ui.TabModel{Title: d.Title()})
	}

	withAll := heapAllocMB()

	// Close them all, one at a time, from the end — the order that leaves a
	// just-closed container in the trailing slot of the backing array.
	for i := tabs - 1; i >= 0; i-- {
		w.drop(i)
	}
	if len(w.docs) != 0 {
		t.Fatalf("expected no tabs left, got %d", len(w.docs))
	}

	afterClose := heapAllocMB()
	// The workspace must stay reachable, or the GC collects it whole and the
	// measurement proves nothing.
	runtime.KeepAlive(w)
	freed := withAll - afterClose
	wantFreed := float64(tabs*fakeBodySize) / 1e6 * 0.9

	t.Logf("%d tabs of %d MB: heap %.1f MB -> %.1f MB (freed %.1f MB)",
		tabs, fakeBodySize>>20, withAll, afterClose, freed)
	if freed < wantFreed {
		t.Errorf("closing every tab freed only %.1f MB of %.1f MB — a closed tab is still reachable",
			freed, float64(tabs*fakeBodySize)/1e6)
	}
}

// TestCloseOneTabReleasesIt covers the single-close case.
func TestCloseOneTabReleasesIt(t *testing.T) {
	w := newWorkspace(func() container.Deps { return container.Deps{} }, nil)
	for i := 0; i < 2; i++ {
		d := newFakeDoc(string(rune('a' + i)))
		w.docs = append(w.docs, d)
		w.tabs = append(w.tabs, ui.TabModel{Title: d.Title()})
	}

	before := heapAllocMB()
	w.drop(1) // close the last tab
	after := heapAllocMB()
	runtime.KeepAlive(w)

	freed := before - after
	want := float64(fakeBodySize) / 1e6 * 0.9
	t.Logf("closing 1 of 2 tabs: heap %.1f MB -> %.1f MB (freed %.1f MB)", before, after, freed)
	if freed < want {
		t.Errorf("closing one tab freed only %.1f MB of %.1f MB — it is still reachable",
			freed, float64(fakeBodySize)/1e6)
	}
}

// TestRealTabOpenCloseReleases is the end-to-end answer: open a real HTTP
// request tab whose body is a large JSON document, close it, and check the Go
// heap comes back. Live heap is the right measure — resident memory can stay
// high because neither Go nor Tree-sitter's C allocator returns pages to the OS
// eagerly, but a closed tab's data must stop being reachable.
func TestRealTabOpenCloseReleases(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	yoga.SetResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})

	body := strings.Repeat(`{"id":1,"name":"item","email":"user@example.com"},`, 120000)
	body = "[" + body + "{}]"
	t.Logf("request body %.1f MB", float64(len(body))/1e6)

	w := newWorkspace(func() container.Deps { return container.Deps{} }, nil)

	open := func() {
		req := &domain.Request{
			MetaData: domain.RequestMeta{ID: "req-1", Name: "big", Type: domain.RequestTypeHTTP},
			Spec: domain.RequestSpec{HTTP: &domain.HTTPRequestSpec{
				Method: "POST",
				Request: &domain.HTTPRequest{
					Body: domain.Body{Type: domain.RequestBodyTypeJSON, Data: body},
				},
			}},
		}
		ct, err := openContainer(container.OpenSpec{Request: req, Deps: container.Deps{}})
		if err != nil {
			t.Fatal(err)
		}
		w.docs = append(w.docs, ct)
		w.tabs = append(w.tabs, ui.TabModel{Title: ct.Title()})
	}

	baseline := heapAllocMB()
	open()
	withTab := heapAllocMB()
	t.Logf("heap: baseline %.1f MB -> tab open %.1f MB (+%.1f MB)",
		baseline, withTab, withTab-baseline)

	w.drop(0)
	afterClose := heapAllocMB()
	runtime.KeepAlive(w)
	runtime.KeepAlive(body)

	held := afterClose - baseline
	grew := withTab - baseline
	t.Logf("heap after close: %.1f MB (still held %.1f MB of the %.1f MB the tab added)",
		afterClose, held, grew)

	// The tab's own data must be gone. Some slack: the request body string
	// itself is kept alive by this test.
	if grew > 1 && held > grew*0.5 {
		t.Errorf("closing the tab released only %.1f MB of %.1f MB — its data is still reachable",
			grew-held, grew)
	}
}
