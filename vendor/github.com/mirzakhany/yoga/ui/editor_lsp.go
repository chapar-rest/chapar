package ui

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/lsp"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// lspManager is the process-sharing language-server manager shared by every
// editor. One server runs per (workspace root, language); see lsp.Manager.
var lspManager = lsp.NewManager()

const (
	hoverDelay        = 500 * time.Millisecond
	completionWidth   = 340
	completionRowH    = 22
	maxCompletionRows = 8
	tooltipMaxLines   = 8
	tooltipPad        = 8
)

// diagSpan is a diagnostic resolved to byte offsets in the current document.
type diagSpan struct {
	lo, hi int
	sev    lsp.DiagnosticSeverity
	msg    string
}

// lspState holds the editor's language-server UI: diagnostics, the completion
// popup, and hover-tooltip tracking. It lives on Editor as lspUI.
type lspState struct {
	diags     []lsp.Diagnostic
	diagSpans []diagSpan

	// Completion popup.
	compOpen   bool
	compItems  []lsp.CompletionItem
	compSel    int
	compFirst  int // index of the first rendered row (simple windowing)
	compAnchor int // byte offset where an accepted item replaces from
	compReqID  int // id of the most recent completion request

	// Hover tooltip.
	hoverText    string     // LSP textDocument/hover reply (async)
	hoverDiags   []diagSpan // diagnostics under the cursor (local, instant)
	hoverReqID   int
	hoverX       float32
	hoverY       float32
	hovMouseX    float32
	hovMouseY    float32
	hovMoved     time.Time
	hovRequested bool
}

func (s *lspState) init() {}

// ---------------------------------------------------------------------------
// Position mapping (byte offsets <-> LSP positions)
// ---------------------------------------------------------------------------

func utf16Len(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

// lspPos converts a byte offset to an LSP position in the negotiated encoding.
func (e *Editor) lspPos(off int) lsp.Position {
	pt := e.pointOf(off) // Row, Col in bytes within the row
	if e.lsp == nil || e.lsp.Encoding() == "utf-8" {
		return lsp.Position{Line: pt.Row, Character: pt.Col}
	}
	line := e.pt.Line(pt.Row)
	if pt.Col > len(line) {
		pt.Col = len(line)
	}
	n := 0
	for _, r := range line[:pt.Col] {
		n += utf16Len(r)
	}
	return lsp.Position{Line: pt.Row, Character: n}
}

// offsetFromLSP converts an LSP position back to a byte offset, clamped to the
// document.
func (e *Editor) offsetFromLSP(p lsp.Position) int {
	row := p.Line
	if row < 0 {
		row = 0
	}
	if lc := e.pt.LineCount(); row >= lc {
		row = lc - 1
	}
	ls := e.pt.LineStart(row)
	line := e.pt.Line(row)
	if e.lsp == nil || e.lsp.Encoding() == "utf-8" {
		col := p.Character
		if col > len(line) {
			col = len(line)
		}
		return ls + col
	}
	n := 0
	for i, r := range line {
		if n >= p.Character {
			return ls + i
		}
		n += utf16Len(r)
	}
	return ls + len(line)
}

// ---------------------------------------------------------------------------
// Synchronization & per-frame polling
// ---------------------------------------------------------------------------

func (e *Editor) lspDidChange() {
	if e.lsp == nil {
		return
	}
	e.lsp.DidChange(e.pt.Bytes())
}

// lspUpdate drains async server events and tracks the hover dwell. Called once
// per frame from Editor.Update.
func (e *Editor) lspUpdate(m *input.Mouse) {
	if e.lsp == nil {
		return
	}
	if evs, ok := e.lsp.Poll(); ok {
		for _, ev := range evs {
			switch ev.Kind {
			case lsp.EventDiagnostics:
				e.lspUI.diags = e.lsp.Diagnostics()
				e.recomputeDiagSpans()
			case lsp.EventCompletion:
				if ev.ReqID == e.lspUI.compReqID {
					e.setCompletions(ev.Completions)
				}
			case lsp.EventHover:
				if ev.ReqID == e.lspUI.hoverReqID && ev.Hover != nil {
					e.lspUI.hoverText = ev.Hover.Text()
				}
			}
		}
	}
	e.trackHover(m)
}

func (e *Editor) recomputeDiagSpans() {
	spans := e.lspUI.diagSpans[:0]
	for _, d := range e.lspUI.diags {
		lo := e.offsetFromLSP(d.Range.Start)
		hi := e.offsetFromLSP(d.Range.End)
		if hi < lo {
			lo, hi = hi, lo
		}
		if hi == lo {
			hi = e.nextRune(lo) // give zero-width diagnostics a visible mark
		}
		spans = append(spans, diagSpan{lo: lo, hi: hi, sev: d.Severity, msg: d.Message})
	}
	e.lspUI.diagSpans = spans
}

// ---------------------------------------------------------------------------
// Hover
// ---------------------------------------------------------------------------

func (e *Editor) trackHover(m *input.Mouse) {
	if m == nil || e.lspUI.compOpen || !e.hoverableAt(m.X, m.Y) {
		e.clearHover()
		return
	}
	if m.X != e.lspUI.hovMouseX || m.Y != e.lspUI.hovMouseY {
		e.lspUI.hovMouseX, e.lspUI.hovMouseY = m.X, m.Y
		e.lspUI.hovMoved = time.Now()
		e.lspUI.hovRequested = false
		e.lspUI.hoverText = ""
		e.lspUI.hoverDiags = nil
		return
	}
	if !e.lspUI.hovRequested && !e.lspUI.hovMoved.IsZero() && time.Since(e.lspUI.hovMoved) >= hoverDelay {
		saved := e.goalX
		off := e.offsetAtPoint(m.X, m.Y)
		e.goalX = saved // offsetAtPoint mutates goalX; hover must not move the caret
		// Diagnostics under the cursor are local data — show them immediately,
		// without waiting for the (async) language-server hover reply.
		e.lspUI.hoverDiags = e.diagnosticsAt(off)
		e.lspUI.hoverReqID = e.lsp.Hover(e.lspPos(off))
		e.lspUI.hovRequested = true
		e.lspUI.hoverX, e.lspUI.hoverY = m.X, m.Y
	}
}

// diagnosticsAt returns the diagnostics whose span covers the byte offset.
func (e *Editor) diagnosticsAt(off int) []diagSpan {
	var out []diagSpan
	for _, sp := range e.lspUI.diagSpans {
		if off >= sp.lo && off < sp.hi {
			out = append(out, sp)
		}
	}
	return out
}

func (e *Editor) hoverableAt(x, y float32) bool {
	vp := e.contentViewport()
	if !vp.Contains(x, y) {
		return false
	}
	return x >= e.viewport.Frame.X+e.gutterW
}

func (e *Editor) clearHover() {
	e.lspUI.hoverText = ""
	e.lspUI.hoverDiags = nil
	e.lspUI.hovMoved = time.Time{}
	e.lspUI.hovRequested = false
}

// lspHoverWake reports the time remaining until a pending hover should fire, so
// AnimationWait can schedule a repaint even while the mouse is stationary.
func (e *Editor) lspHoverWake() (time.Duration, bool) {
	if e.lspUI.hovMoved.IsZero() || e.lspUI.hovRequested {
		return 0, false
	}
	rem := hoverDelay - time.Since(e.lspUI.hovMoved)
	if rem < 0 {
		rem = 0
	}
	return rem, true
}

// ---------------------------------------------------------------------------
// Completion
// ---------------------------------------------------------------------------

func isIdentRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (e *Editor) identStart(off int) int {
	b := e.pt.Bytes()
	for off > 0 {
		r, sz := utf8.DecodeLastRune(b[:off])
		if !isIdentRune(r) {
			break
		}
		off -= sz
	}
	return off
}

// lspAfterType opens or refilters completion after a text-producing keystroke.
func (e *Editor) lspAfterType(runes []rune) {
	if e.lsp == nil || !e.focused {
		return
	}
	if len(runes) != 1 {
		e.closeCompletion()
		return
	}
	switch r := runes[0]; {
	case r == '.':
		e.requestCompletion()
	case isIdentRune(r):
		if e.lspUI.compOpen {
			e.requestCompletion()
		}
	default:
		e.closeCompletion()
	}
}

func (e *Editor) requestCompletion() {
	if e.lsp == nil {
		return
	}
	e.lspUI.compAnchor = e.identStart(e.caret)
	e.lspUI.compReqID = e.lsp.Completion(e.lspPos(e.caret))
	e.markParsePending()
}

func (e *Editor) setCompletions(items []lsp.CompletionItem) {
	prefix := strings.ToLower(e.completionPrefix())
	filtered := items[:0:0]
	for _, it := range items {
		ft := it.FilterText
		if ft == "" {
			ft = it.Label
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(ft), prefix) {
			filtered = append(filtered, it)
		}
	}
	if len(filtered) == 0 {
		e.closeCompletion()
		return
	}
	e.lspUI.compItems = filtered
	e.lspUI.compSel = 0
	e.lspUI.compFirst = 0
	e.lspUI.compOpen = true
}

// completionPrefix is the identifier text between the anchor and the caret.
func (e *Editor) completionPrefix() string {
	lo, hi := e.lspUI.compAnchor, e.caret
	b := e.pt.Bytes()
	if lo < 0 {
		lo = 0
	}
	if hi > len(b) {
		hi = len(b)
	}
	if lo > hi {
		return ""
	}
	return string(b[lo:hi])
}

func (e *Editor) closeCompletion() {
	e.lspUI.compOpen = false
	e.lspUI.compItems = nil
	e.lspUI.compSel = 0
	e.lspUI.compFirst = 0
}

// handleCompletionKey processes a key while the popup is open. It returns true
// when the key was consumed by the popup.
func (e *Editor) handleCompletionKey(ev input.KeyEvent) bool {
	n := len(e.lspUI.compItems)
	if n == 0 {
		e.closeCompletion()
		return false
	}
	switch ev.Key {
	case input.KeyUp:
		e.lspUI.compSel = (e.lspUI.compSel - 1 + n) % n
		e.ensureCompletionVisible()
		return true
	case input.KeyDown:
		e.lspUI.compSel = (e.lspUI.compSel + 1) % n
		e.ensureCompletionVisible()
		return true
	case input.KeyEnter, input.KeyTab:
		e.acceptCompletion()
		return true
	case input.KeyEscape:
		e.closeCompletion()
		return true
	case input.KeyBackspace:
		e.backspace()
		if e.caret <= e.lspUI.compAnchor {
			e.closeCompletion()
		} else {
			e.requestCompletion()
		}
		return true
	case input.KeyLeft, input.KeyRight, input.KeyHome, input.KeyEnd:
		e.closeCompletion()
		return false
	}
	return false
}

func (e *Editor) ensureCompletionVisible() {
	if e.lspUI.compSel < e.lspUI.compFirst {
		e.lspUI.compFirst = e.lspUI.compSel
	}
	if e.lspUI.compSel >= e.lspUI.compFirst+maxCompletionRows {
		e.lspUI.compFirst = e.lspUI.compSel - maxCompletionRows + 1
	}
}

func (e *Editor) acceptCompletion() {
	sel := e.lspUI.compSel
	if sel < 0 || sel >= len(e.lspUI.compItems) {
		e.closeCompletion()
		return
	}
	it := e.lspUI.compItems[sel]
	lo := e.lspUI.compAnchor
	if lo > e.caret {
		lo = e.caret
	}
	e.closeCompletion()
	e.applyEdit(lo, e.caret-lo, it.Insert(), false)
}

// ---------------------------------------------------------------------------
// Diagnostics painting (called from Editor.paint per line)
// ---------------------------------------------------------------------------

func (e *Editor) paintDiagnosticsLine(dl *render.DrawList, ls, lineEnd int, sline shape.Line, drawX float32, winLo, winHi int, y float32) {
	if len(e.lspUI.diagSpans) == 0 {
		return
	}
	th := theme.Current()
	m := frameText().MetricsMono()
	textH := m.Ascent + m.Descent
	vpad := (e.lineH - textH) / 2
	underY := y + vpad + textH - 1
	for _, sp := range e.lspUI.diagSpans {
		if sp.hi <= ls || sp.lo >= lineEnd {
			continue
		}
		a := maxInt(sp.lo, ls)
		b := minInt(sp.hi, lineEnd)
		col := th.Error
		if sp.sev >= lsp.SeverityWarning {
			col = th.Warning
		}
		for _, rect := range e.windowRects(sline, a-ls, b-ls, winLo, winHi, y, e.lineH) {
			e.paintSquiggle(dl, drawX+rect[0], underY, rect[2], col)
		}
	}
}

// paintSquiggle draws a small dotted wave under a span as an error/warning mark.
func (e *Editor) paintSquiggle(dl *render.DrawList, x, y, w float32, col render.Color) {
	const step = 3
	n := int(w / step)
	for i := 0; i < n; i++ {
		dy := float32(0)
		if i%2 == 1 {
			dy = 2
		}
		dl.AddRect(render.Rect{X: x + float32(i)*step, Y: y + dy, W: 2, H: 2}, col)
	}
}

// ---------------------------------------------------------------------------
// Overlay rendering (completion popup + hover tooltip)
//
// These run from a host-owned overlay element so the popup paints on top of and
// outside the clipped editor viewport, the way Menu does.
// ---------------------------------------------------------------------------

// completionRect is the screen rectangle of the completion popup.
func (e *Editor) completionRect() render.Rect {
	r := e.rowOfByte(e.lspUI.compAnchor)
	x0 := e.viewport.Frame.X + e.gutterW + e.textPad - e.ScrollX
	cx := x0 + e.xInRow(e.lspUI.compAnchor)
	cy := e.viewport.Frame.Y + float32(r+1)*e.lineH - e.ScrollPx
	rows := minInt(len(e.lspUI.compItems), maxCompletionRows)
	w := float32(completionWidth)
	h := float32(rows) * completionRowH
	cx, cy = clampToViewport(cx, cy, w, h)
	return render.Rect{X: cx, Y: cy, W: w, H: h}
}

// PaintLSPOverlay draws the completion popup and hover tooltip. It is wired as
// the Paint hook of a host overlay element (see the example's EditorPage).
func (e *Editor) PaintLSPOverlay(dl *render.DrawList, eng *shape.Engine) {
	if e.lspUI.compOpen {
		e.paintCompletion(dl, eng)
	} else if len(e.lspUI.hoverDiags) > 0 || strings.TrimSpace(e.lspUI.hoverText) != "" {
		e.paintTooltip(dl, eng)
	}
}

func (e *Editor) paintCompletion(dl *render.DrawList, eng *shape.Engine) {
	th := theme.Current()
	r := e.completionRect()
	dl.AddElevationShadow(r, 6, render.Shadow{OffsetY: 2, Blur: 12, Color: render.Color{A: 1}})
	dl.AddRoundedRectBorder(r, 6, 1, th.Chrome, th.Border)
	dl.PushClip(r)
	first := e.lspUI.compFirst
	last := minInt(first+maxCompletionRows, len(e.lspUI.compItems))
	for i := first; i < last; i++ {
		it := e.lspUI.compItems[i]
		row := render.Rect{X: r.X, Y: r.Y + float32(i-first)*completionRowH, W: r.W, H: completionRowH}
		if i == e.lspUI.compSel {
			dl.AddRoundedRect(row, 4, th.ListActive)
		}
		ty := row.Y + (completionRowH-eng.MetricsMono().LineHeight)/2
		eng.DrawStringTopMono(dl, it.Label, row.X+8, ty, th.Foreground)
		if it.Detail != "" {
			dw, _ := eng.MeasureMono(it.Detail)
			detail := it.Detail
			if dw > r.W*0.5 {
				detail = "" // skip detail that would overrun the row
			}
			if detail != "" {
				eng.DrawStringTopMono(dl, detail, row.X+r.W-dw-8, ty, th.ForegroundMuted)
			}
		}
	}
	dl.PopClip()
}

// tipLine is one display row of the hover tooltip with its color.
type tipLine struct {
	text string
	col  render.Color
}

func (e *Editor) paintTooltip(dl *render.DrawList, eng *shape.Engine) {
	th := theme.Current()
	lines := e.tooltipLines(th)
	if len(lines) == 0 {
		return
	}
	lineH := eng.MetricsMono().LineHeight
	var maxW float32
	for _, ln := range lines {
		if w, _ := eng.MeasureMono(ln.text); w > maxW {
			maxW = w
		}
	}
	w := maxW + tooltipPad*2
	h := float32(len(lines))*lineH + tooltipPad*2
	x, y := clampToViewport(e.lspUI.hoverX+12, e.lspUI.hoverY+18, w, h)
	r := render.Rect{X: x, Y: y, W: w, H: h}
	dl.AddElevationShadow(r, 6, render.Shadow{OffsetY: 2, Blur: 12, Color: render.Color{A: 1}})
	dl.AddRoundedRectBorder(r, 6, 1, th.Chrome, th.Border)
	dl.PushClip(r)
	for i, ln := range lines {
		eng.DrawStringTopMono(dl, ln.text, r.X+tooltipPad, r.Y+tooltipPad+float32(i)*lineH, ln.col)
	}
	dl.PopClip()
}

// tooltipLines builds the tooltip content: diagnostic messages (colored by
// severity) on top, then the language-server hover info beneath a blank gap.
func (e *Editor) tooltipLines(th *theme.Theme) []tipLine {
	var lines []tipLine
	for _, d := range e.lspUI.hoverDiags {
		col := th.Error
		if d.sev >= lsp.SeverityWarning {
			col = th.Warning
		}
		for _, ln := range strings.Split(d.msg, "\n") {
			lines = append(lines, tipLine{text: strings.TrimRight(ln, " \t"), col: col})
		}
	}
	hover := cleanHover(e.lspUI.hoverText)
	if len(lines) > 0 && len(hover) > 0 {
		lines = append(lines, tipLine{}) // blank separator row
	}
	for _, ln := range hover {
		lines = append(lines, tipLine{text: ln, col: th.Foreground})
	}
	return lines
}

// cleanHover turns a markdown hover payload into a few plain display lines.
func cleanHover(s string) []string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimRight(ln, " \t")
		if strings.HasPrefix(strings.TrimSpace(ln), "```") {
			continue // drop code fences
		}
		if strings.TrimSpace(ln) == "" && len(out) == 0 {
			continue
		}
		out = append(out, ln)
		if len(out) >= tooltipMaxLines {
			break
		}
	}
	// Trim trailing blank lines.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return out
}

// LSPOverlayMouse handles clicks on the completion popup. It is wired as the
// OnMouse hook of the host overlay element.
func (e *Editor) LSPOverlayMouse(_ *layout.Element, m *input.Mouse) {
	if !e.lspUI.compOpen || m == nil {
		return
	}
	r := e.completionRect()
	if !r.Contains(m.X, m.Y) {
		if m.Pressed {
			e.closeCompletion()
		}
		return
	}
	m.Consumed = true
	idx := e.lspUI.compFirst + int((m.Y-r.Y)/completionRowH)
	if idx < 0 || idx >= len(e.lspUI.compItems) {
		return
	}
	e.lspUI.compSel = idx
	if m.Released {
		e.acceptCompletion()
	}
}
