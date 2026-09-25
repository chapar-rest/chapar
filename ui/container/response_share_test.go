package container_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"

	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/util"
	"github.com/chapar-rest/chapar/ui/container"
)

func setupText(t *testing.T) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	yoga.SetResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
}

// formattedJSON mimics a server that already pretty-prints, which is how the
// large payloads that caused the problem are usually served.
func formattedJSON(n int) []byte {
	var b strings.Builder
	b.Grow(n + 64)
	b.WriteString("[\n")
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(&b, "  {\n    \"login\": \"user%06d\",\n    \"commits\": %d\n  },\n", i, i)
	}
	b.WriteString("  {}\n]\n")
	return []byte(b.String())
}

// TestResponseEditorSharesBodyBuffer is the regression guard for per-tab
// memory: opening a large response used to cost several full copies of it (a
// formatted string, the display slice, the piece table's original, and the
// piece table's flat cache). The editor must now read the response's own bytes.
func TestResponseEditorSharesBodyBuffer(t *testing.T) {
	setupText(t)

	raw := formattedJSON(4 << 20)
	kind, pretty, jsonStr, isJSON := util.ApplyBodyFormat("application/json", raw)
	if pretty != "" {
		t.Fatalf("an already-formatted body should not be reformatted (got %d bytes)", len(pretty))
	}
	res := &egress.Response{
		Body: raw, BodyKind: kind, Pretty: pretty, JSON: jsonStr, IsJSON: isJSON, Size: len(raw),
	}

	display := container.DisplayBody(res, false)
	if unsafe.SliceData(display) != unsafe.SliceData(raw) {
		t.Error("DisplayBody copied the body instead of returning it")
	}

	ed := container.ReplaceResponseEditor(nil, res, false)
	defer ed.Close()

	if got := ed.Bytes(); unsafe.SliceData(got) != unsafe.SliceData(raw) {
		t.Error("the editor copied the response body instead of reading it in place")
	}
	if ed.Bytes()[0] != '[' {
		t.Errorf("editor content starts with %q", ed.Bytes()[0])
	}
}

// TestResponseEditorPerTabCost bounds what one tab adds on top of the body it
// displays. Before, a tab cost several multiples of the body.
func TestResponseEditorPerTabCost(t *testing.T) {
	setupText(t)

	raw := formattedJSON(8 << 20)
	kind, pretty, jsonStr, isJSON := util.ApplyBodyFormat("application/json", raw)
	res := &egress.Response{
		Body: raw, BodyKind: kind, Pretty: pretty, JSON: jsonStr, IsJSON: isJSON, Size: len(raw),
	}

	heap := func() uint64 {
		runtime.GC()
		runtime.GC()
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return ms.HeapAlloc
	}

	before := heap()
	ed := container.ReplaceResponseEditor(nil, res, false)
	added := heap() - before

	// A tab's own cost is its line index — one int per line — not another copy
	// of the document. Budget that plus slack for the editor's fixed state.
	lines := strings.Count(string(raw), "\n") + 1
	budget := uint64(lines*int(unsafe.Sizeof(int(0)))) + 2<<20
	if added > budget {
		t.Errorf("one tab added %.1f MB for a %.1f MB body (%d lines, budget %.1f MB) — looks like a copy",
			float64(added)/1e6, float64(len(raw))/1e6, lines, float64(budget)/1e6)
	}
	t.Logf("%.1f MB body, %d lines -> tab added %.1f MB (budget %.1f MB)",
		float64(len(raw))/1e6, lines, float64(added)/1e6, float64(budget)/1e6)
	ed.Close()
	runtime.KeepAlive(res)
}

// TestRawToggleAlsoShares checks the other display path.
func TestRawToggleAlsoShares(t *testing.T) {
	setupText(t)

	raw := formattedJSON(1 << 20)
	kind, pretty, jsonStr, isJSON := util.ApplyBodyFormat("application/json", raw)
	res := &egress.Response{
		Body: raw, BodyKind: kind, Pretty: pretty, JSON: jsonStr, IsJSON: isJSON, Size: len(raw),
	}

	ed := container.ReplaceResponseEditor(nil, res, true)
	defer ed.Close()
	if unsafe.SliceData(ed.Bytes()) != unsafe.SliceData(raw) {
		t.Error("raw view copied the body instead of reading it in place")
	}
}
