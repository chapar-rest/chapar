package ui

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// The find and replace bar. Its two inputs are real TextInputs, hosted and
// painted by the editor, so selection, word and line clicks, clipboard and
// undo inside them behave like any other text field. The editor keeps
// keyboard focus; while the bar is focused it hands keys and text to the
// focused input after the bar's own shortcuts (ScopeFind in the keymap).

// findHit names a part of the bar, for hit-testing, painting and Tab order.
type findHit int

const (
	hitNone findHit = iota
	hitChevron
	hitQuery
	hitCase
	hitWord
	hitRegex
	hitPrev
	hitNext
	hitInSel
	hitClose
	hitReplace
	hitPreserve
	hitReplOne
	hitReplAll
	hitBar // the bar's background
	hitCount
)

// searchState is the find bar's state.
type searchState struct {
	open        bool
	focused     bool // the bar, not the text area, takes keys
	replaceMode bool
	focus       findHit // the input or button keys go to

	query, repl *TextInput

	caseSens, wholeWord, regex, preserveCase bool

	inSel        bool // search only selLo..selHi
	selLo, selHi int

	matches  []matchRange
	current  int  // index of the active match, -1 for none
	capped   bool // more matches than findMatchCap
	badRegex bool

	from      int       // offset the live search looks forward from
	pending   bool      // a debounced search is due
	revealDue bool      // and should select its match when it runs
	due       time.Time // when the debounced search runs
	replacing bool      // an edit is the bar's own replace
	seeding   bool      // the query is being filled from the selection

	reKey string
	re    *regexp.Regexp
	reErr error

	hover   findHit
	hoverAt time.Time
}

// matchRange is a [lo, hi) byte range in the document.
type matchRange struct{ lo, hi int }

const (
	findMatchCap      = 10000     // matches counted and highlighted
	findDebounceBytes = 256 << 10 // documents larger than this search after a pause
	findDebounce      = 120 * time.Millisecond
	findTipDelay      = 500 * time.Millisecond

	findRowH     = float32(34)
	findFieldH   = float32(26)
	findPad      = float32(4)
	findChevW    = float32(20)
	findToggleW  = float32(22)
	findBtnW     = float32(26)
	findCountW   = float32(78)
	findFieldMin = float32(150)
	findFieldMax = float32(300)
	findIconSz   = float32(16)
)

// OpenFind opens the in-file find bar, or find and replace when replace is
// set, seeded with the selection. Apps that move the bar off ⌘F call it from
// their own command.
func (e *Editor) OpenFind(replace bool) { e.openSearch(replace) }

// FindFocused reports whether the find bar has the keyboard, so an app can
// keep its own editor shortcuts from acting on the document meanwhile.
func (e *Editor) FindFocused() bool { return e.findFocused() }

// FindQuery returns the find field's text.
func (e *Editor) FindQuery() string {
	if e.search.query == nil {
		return ""
	}
	return e.search.query.Value
}

// FindMatches returns how many matches the find bar shows, and the 1-based
// index of the active one (0 for none).
func (e *Editor) FindMatches() (current, total int) {
	return e.search.current + 1, len(e.search.matches)
}

func (e *Editor) findFocused() bool { return e.search.open && e.search.focused }

// findInput returns the focused find bar input, or nil.
func (e *Editor) findInput() *TextInput {
	if !e.findFocused() {
		return nil
	}
	switch e.search.focus {
	case hitQuery:
		return e.search.query
	case hitReplace:
		return e.search.repl
	}
	return nil
}

func (e *Editor) ensureFind() {
	if e.search.query != nil {
		return
	}
	e.search.query = NewTextInput(TextFieldConfig{Placeholder: "Find"})
	e.search.repl = NewTextInput(TextFieldConfig{Placeholder: "Replace"})
	e.search.query.OnChange = func(string) {
		if !e.search.seeding {
			e.scheduleSearch(e.search.from, true)
		}
	}
	e.search.current = -1
}

func (e *Editor) openSearch(replace bool) {
	e.ensureFind()
	s := &e.search
	wasOpen := s.open
	s.open, s.focused = true, true
	if replace && !e.readOnly {
		s.replaceMode = true
	} else if !wasOpen {
		s.replaceMode = false
	}
	lo, hi := e.selRange()
	s.from = lo
	s.inSel = false
	if lo < hi {
		if sel := e.pt.Bytes()[lo:hi]; bytes.IndexByte(sel, '\n') < 0 {
			q := s.query
			s.seeding = true // the selection is the match; nothing to jump to
			q.edit(mergeNone, "", func() { q.setValue(string(sel)) })
			s.seeding = false
		} else {
			// A multi-line selection is where to look, not what to look for.
			s.inSel, s.selLo, s.selHi = true, lo, hi
		}
	}
	focus := hitQuery
	if replace && s.replaceMode && s.query.Value != "" && wasOpen {
		focus = hitReplace
	}
	e.focusFind(focus, true)
	s.pending = false
	e.runSearch(s.from)
}

// closeSearch closes the bar.
func (e *Editor) closeSearch() {
	s := &e.search
	s.open, s.focused, s.pending = false, false, false
	s.matches = s.matches[:0]
	s.current = -1
	s.inSel = false
	s.hover = hitNone
	e.findBlur()
}

// closeSearchToMatch closes the bar with the active match selected, and
// gives the keyboard back to the text.
func (e *Editor) closeSearchToMatch() {
	if e.search.pending {
		e.runSearch(e.search.from)
	}
	if c := e.search.current; c >= 0 && c < len(e.search.matches) {
		e.jumpToMatch(c)
	}
	e.closeSearch()
}

// focusFind moves the bar's keyboard focus to h. With selectAll an input
// selects its text, as Tab into it does.
func (e *Editor) focusFind(h findHit, selectAll bool) {
	e.search.focus = h
	e.syncFindFocus()
	if f := e.findInput(); f != nil && selectAll {
		f.SelectAll()
	}
	e.blinkStart = time.Now()
}

// syncFindFocus focuses the input the bar's focus names and blurs the other.
func (e *Editor) syncFindFocus() {
	s := &e.search
	if s.query == nil {
		return
	}
	for h, f := range map[findHit]*TextInput{hitQuery: s.query, hitReplace: s.repl} {
		want := s.open && s.focused && s.focus == h
		if want && !f.focused {
			f.Focus()
		} else if !want && f.focused {
			f.Blur()
		}
	}
}

func (e *Editor) findBlur() {
	if e.search.query != nil {
		e.search.query.Blur()
		e.search.repl.Blur()
	}
}

// findLayout runs the inputs' per-frame upkeep.
func (e *Editor) findLayout(c *Ctx) {
	if e.search.query == nil {
		return
	}
	if e.focused {
		e.syncFindFocus()
	}
	for _, f := range []*TextInput{e.search.query, e.search.repl} {
		f.Update(nil)
		f.menu.layout(c)
	}
}

// findUpdate runs a debounced search once it is due.
func (e *Editor) findUpdate() {
	s := &e.search
	if s.open && s.pending && !time.Now().Before(s.due) {
		s.pending = false
		e.runSearch(s.from)
		if s.revealDue {
			e.revealMatch()
		}
	}
}

// findWake is how soon the bar needs a frame: for a debounced search, or to
// show a tooltip.
func (e *Editor) findWake() (time.Duration, bool) {
	s := &e.search
	if !s.open {
		return 0, false
	}
	var at time.Time
	switch {
	case s.pending:
		at = s.due
	case s.hover != hitNone && s.hover != hitBar && time.Since(s.hoverAt) < findTipDelay:
		at = s.hoverAt.Add(findTipDelay)
	default:
		return 0, false
	}
	return max(time.Until(at), time.Millisecond), true
}

// findAfterEdit keeps the bar current after the document changes.
func (e *Editor) findAfterEdit(edit highlight.Edit) {
	s := &e.search
	if s.inSel {
		d := edit.NewEndByte - edit.OldEndByte
		switch {
		case edit.OldEndByte <= s.selLo:
			s.selLo += d
			s.selHi += d
		case edit.StartByte <= s.selHi:
			s.selHi = max(s.selLo, s.selHi+d)
		}
	}
	if s.replacing {
		return // the replace runs its own search
	}
	lo, _ := e.selRange()
	e.scheduleSearch(lo, false)
}

// scheduleSearch searches from offset from now, or after a pause on a large
// document. reveal selects the match it finds.
func (e *Editor) scheduleSearch(from int, reveal bool) {
	s := &e.search
	if !s.open {
		return
	}
	if e.pt.Len() > findDebounceBytes {
		s.pending, s.revealDue = true, reveal || (s.pending && s.revealDue)
		s.from = from
		s.due = time.Now().Add(findDebounce)
		return
	}
	s.pending = false
	e.runSearch(from)
	if reveal {
		e.revealMatch()
	}
}

// findRegexp compiles the query with the bar's options. literal reports a
// plain, case-sensitive string, which is searched without a regexp.
func (e *Editor) findRegexp() (re *regexp.Regexp, literal bool, err error) {
	s := &e.search
	q := s.query.Value
	if !s.regex && !s.wholeWord && s.caseSens {
		return nil, true, nil
	}
	pat := q
	if !s.regex {
		pat = regexp.QuoteMeta(q)
	}
	if s.wholeWord {
		first, _ := utf8.DecodeRuneInString(q)
		last, _ := utf8.DecodeLastRuneInString(q)
		switch {
		case s.regex:
			pat = `\b(?:` + pat + `)\b`
		default:
			if isWordRune(first) {
				pat = `\b` + pat
			}
			if isWordRune(last) {
				pat += `\b`
			}
		}
	}
	flags := "(?m)"
	if !s.caseSens {
		flags = "(?mi)"
	}
	pat = flags + pat
	if pat == s.reKey {
		return s.re, false, s.reErr
	}
	s.reKey = pat
	s.re, s.reErr = regexp.Compile(pat)
	return s.re, false, s.reErr
}

func isWordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// findRange is the part of the document the bar searches.
func (e *Editor) findRange() (lo, hi int) {
	if e.search.inSel {
		n := e.pt.Len()
		return clampInt(e.search.selLo, 0, n), clampInt(e.search.selHi, 0, n)
	}
	return 0, e.pt.Len()
}

// findAll returns up to limit matches (all with limit < 0) as absolute
// offsets, each a pair for a literal search or regexp submatch indexes.
func (e *Editor) findAll(limit int) ([][]int, error) {
	q := e.search.query.Value
	if q == "" {
		return nil, nil
	}
	lo, hi := e.findRange()
	doc := e.pt.Bytes()[lo:hi]
	re, literal, err := e.findRegexp()
	if err != nil {
		return nil, err
	}
	var out [][]int
	if literal {
		needle := []byte(q)
		for at := 0; at+len(needle) <= len(doc) && (limit < 0 || len(out) < limit); {
			i := bytes.Index(doc[at:], needle)
			if i < 0 {
				break
			}
			pos := at + i
			out = append(out, []int{lo + pos, lo + pos + len(needle)})
			at = pos + len(needle)
		}
		return out, nil
	}
	for _, m := range re.FindAllSubmatchIndex(doc, limit) {
		if m[0] == m[1] {
			continue // an empty match (such as ^) selects nothing
		}
		for k := range m {
			if m[k] >= 0 {
				m[k] += lo
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// runSearch recomputes the matches; the active one is the first at or after
// from.
func (e *Editor) runSearch(from int) {
	s := &e.search
	s.matches = s.matches[:0]
	s.capped, s.badRegex = false, false
	s.current = -1
	if s.query == nil || s.query.Value == "" {
		return
	}
	all, err := e.findAll(findMatchCap + 1)
	if err != nil {
		s.badRegex = true
		return
	}
	if len(all) > findMatchCap {
		all, s.capped = all[:findMatchCap], true
	}
	for _, m := range all {
		s.matches = append(s.matches, matchRange{m[0], m[1]})
	}
	for i, mr := range s.matches {
		if mr.lo >= from {
			s.current = i
			break
		}
	}
	if s.current < 0 && len(s.matches) > 0 {
		s.current = 0
	}
}

// revealMatch selects the active match and scrolls to it.
func (e *Editor) revealMatch() {
	if c := e.search.current; c >= 0 {
		e.jumpToMatch(c)
	}
}

func (e *Editor) flushSearch() {
	if e.search.pending {
		e.search.pending = false
		e.runSearch(e.search.from)
	}
}

func (e *Editor) searchNext() { e.searchStep(1) }

func (e *Editor) searchPrev() { e.searchStep(-1) }

// searchStep moves to the next (dir 1) or previous (dir -1) match. When the
// selection is not the active match, as after moving the caret, it goes to
// the match after (or before) the selection instead.
func (e *Editor) searchStep(dir int) {
	e.flushSearch()
	ms := e.search.matches
	n := len(ms)
	if n == 0 {
		return
	}
	cur := e.search.current
	lo, hi := e.selRange()
	if cur >= 0 && cur < n && ms[cur].lo == lo && ms[cur].hi == hi {
		cur = (cur + dir + n) % n
	} else if dir > 0 {
		cur = 0
		for i, mr := range ms {
			if mr.lo >= hi {
				cur = i
				break
			}
		}
	} else {
		cur = n - 1
		for i := n - 1; i >= 0; i-- {
			if ms[i].hi <= lo {
				cur = i
				break
			}
		}
	}
	e.search.current = cur
	e.search.from = ms[cur].lo
	e.jumpToMatch(cur)
}

func (e *Editor) jumpToMatch(idx int) {
	if idx < 0 || idx >= len(e.search.matches) {
		return
	}
	mr := e.search.matches[idx]
	e.extra = nil
	e.selAnchor = mr.lo
	e.caret = mr.hi
	e.lastMerge = mergeNone
	e.ensureCaretVisible()
}

// replacementFor is the replace text for one match: with regex on, $1,
// ${name} and $& refer to groups, and \n and \t are a newline and a tab.
func (e *Editor) replacementFor(m []int) string {
	s := &e.search
	tmpl := s.repl.Value
	doc := e.pt.Bytes()
	out := tmpl
	if s.regex && s.re != nil {
		out = string(s.re.Expand(nil, []byte(goTemplate(tmpl)), doc, m))
	}
	if s.preserveCase {
		out = matchCase(string(doc[m[0]:m[1]]), out)
	}
	return out
}

var dollarGroup = regexp.MustCompile(`\$(\d+)`)

// goTemplate turns the usual replace syntax into Go's Expand template:
// "$1x" would name group "1x" there, so numbered groups get braces.
func goTemplate(t string) string {
	var b strings.Builder
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c == '\\' && i+1 < len(t) {
			switch t[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case 't':
				b.WriteByte('\t')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		if c == '$' && i+1 < len(t) && t[i+1] == '&' {
			b.WriteString("${0}")
			i++
			continue
		}
		b.WriteByte(c)
	}
	return dollarGroup.ReplaceAllString(b.String(), "$${$1}")
}

// matchCase gives rep the case pattern of matched: all upper, all lower, or
// a leading capital.
func matchCase(matched, rep string) string {
	hasLetter := strings.IndexFunc(matched, unicode.IsLetter) >= 0
	switch {
	case !hasLetter:
		return rep
	case strings.ToUpper(matched) == matched:
		return strings.ToUpper(rep)
	case strings.ToLower(matched) == matched:
		return strings.ToLower(rep)
	}
	first, _ := utf8.DecodeRuneInString(matched)
	if unicode.IsUpper(first) {
		r, sz := utf8.DecodeRuneInString(rep)
		return string(unicode.ToUpper(r)) + rep[sz:]
	}
	return rep
}

// currentMatchSubs returns the active match with its submatch indexes.
func (e *Editor) currentMatchSubs() []int {
	s := &e.search
	mr := s.matches[s.current]
	re, literal, err := e.findRegexp()
	if literal || err != nil {
		return []int{mr.lo, mr.hi}
	}
	_, hi := e.findRange()
	m := re.FindSubmatchIndex(e.pt.Bytes()[mr.lo:hi])
	if m == nil || m[0] != 0 {
		return []int{mr.lo, mr.hi}
	}
	for k := range m {
		if m[k] >= 0 {
			m[k] += mr.lo
		}
	}
	return m
}

// doReplace replaces the active match and moves to the next one.
func (e *Editor) doReplace() {
	s := &e.search
	if e.readOnly || !s.replaceMode {
		return
	}
	e.flushSearch()
	if s.current < 0 || s.current >= len(s.matches) {
		return
	}
	m := e.currentMatchSubs()
	rep := e.replacementFor(m)
	s.replacing = true
	e.extra = nil
	e.applyEdit(m[0], m[1]-m[0], rep, mergeNone)
	s.replacing = false
	e.runSearch(m[0] + len(rep))
	e.revealMatch()
}

// doReplaceAll replaces every match as one edit, so one undo step.
func (e *Editor) doReplaceAll() {
	s := &e.search
	if e.readOnly || !s.replaceMode || s.query == nil {
		return
	}
	all, err := e.findAll(-1)
	if err != nil || len(all) == 0 {
		return
	}
	doc := e.pt.Bytes()
	first, last := all[0][0], all[len(all)-1][1]
	var b strings.Builder
	prev := first
	for _, m := range all {
		b.Write(doc[prev:m[0]])
		b.WriteString(e.replacementFor(m))
		prev = m[1]
	}
	s.replacing = true
	e.extra = nil
	e.UndoGroup(func() { e.applyEdit(first, last-first, b.String(), mergeNone) })
	s.replacing = false
	e.runSearch(e.caret)
}

func (e *Editor) toggleFind(flag *bool) {
	*flag = !*flag
	e.scheduleSearch(e.search.from, false)
}

// toggleInSelection limits the search to the selection, or lifts the limit.
func (e *Editor) toggleInSelection() {
	s := &e.search
	if s.inSel {
		s.inSel = false
	} else if lo, hi := e.selRange(); lo < hi {
		s.inSel, s.selLo, s.selHi = true, lo, hi
	} else {
		return
	}
	e.scheduleSearch(s.from, false)
}

func (e *Editor) toggleReplaceRow() {
	s := &e.search
	if e.readOnly {
		return
	}
	s.replaceMode = !s.replaceMode
	if !s.replaceMode && s.focus >= hitReplace {
		e.focusFind(hitQuery, false)
	}
}

// findStops is the bar's Tab order.
func (e *Editor) findStops() []findHit {
	stops := []findHit{hitQuery, hitCase, hitWord, hitRegex, hitPrev, hitNext, hitInSel, hitClose}
	if !e.readOnly {
		stops = append([]findHit{hitChevron}, stops...)
	}
	if e.search.replaceMode {
		stops = append(stops, hitReplace, hitPreserve, hitReplOne, hitReplAll)
	}
	return stops
}

func (e *Editor) findTab(dir int) {
	stops := e.findStops()
	at := 0
	for i, h := range stops {
		if h == e.search.focus {
			at = i
		}
	}
	n := len(stops)
	e.focusFind(stops[(at+dir+n)%n], true)
}

// findActivate does what clicking (or Space on) part of the bar does.
func (e *Editor) findActivate(h findHit) {
	s := &e.search
	switch h {
	case hitChevron:
		e.toggleReplaceRow()
	case hitCase:
		e.toggleFind(&s.caseSens)
	case hitWord:
		e.toggleFind(&s.wholeWord)
	case hitRegex:
		e.toggleFind(&s.regex)
	case hitPreserve:
		s.preserveCase = !s.preserveCase
	case hitPrev:
		e.searchPrev()
	case hitNext:
		e.searchNext()
	case hitInSel:
		e.toggleInSelection()
	case hitClose:
		e.closeSearchToMatch()
	case hitReplOne:
		e.doReplace()
	case hitReplAll:
		e.doReplaceAll()
	}
}

// findHandleKey handles a key while the bar is focused: the bar's shortcuts
// first, then the focused input, or Enter and Space on a focused button.
func (e *Editor) findHandleKey(ev input.KeyEvent) {
	scope := ScopeFind
	switch e.search.focus {
	case hitQuery:
		scope = ScopeFindQuery
	case hitReplace:
		scope = ScopeFindReplace
	}
	if ev.Key == input.KeyTab && !ev.Mods.Primary() && !ev.Mods.Has(input.ModAlt) {
		dir := 1
		if ev.Mods.Has(input.ModShift) {
			dir = -1
		}
		e.findTab(dir)
		return
	}
	if e.runKey(ev, scope) {
		return
	}
	if f := e.findInput(); f != nil {
		f.HandleKeys([]input.KeyEvent{ev})
		e.blinkStart = time.Now()
		return
	}
	if (ev.Key == input.KeyEnter || ev.Key == input.KeySpace) && ev.Mods == 0 {
		e.findActivate(e.search.focus)
	}
}

func (e *Editor) findHandleText(runes []rune) {
	if f := e.findInput(); f != nil {
		f.HandleText(runes)
		e.blinkStart = time.Now()
	}
}

// findGeom is where each part of the bar sits this frame.
type findGeom struct {
	bar render.Rect
	r   [hitCount]render.Rect
}

func (e *Editor) findGeom() findGeom {
	var g findGeom
	f := e.viewport.Frame
	_, _, vShow, _ := e.scrollMetrics()
	vbarW := float32(0)
	if vShow {
		vbarW = editorBarSize
	}
	fixed := 2*findPad + findChevW + 2 + 6 + findCountW + 4*(findBtnW+2)
	fieldW := clampf(f.W-vbarW-12-fixed, findFieldMin, findFieldMax)
	rows := float32(1)
	if e.search.replaceMode {
		rows = 2
	}
	w := fixed + fieldW
	g.bar = render.Rect{X: max(f.X+f.W-vbarW-6-w, f.X), Y: f.Y + 6, W: w, H: rows*findRowH + findPad}
	rowY := func(i int) float32 { return g.bar.Y + findPad/2 + float32(i)*findRowH }
	fieldY := func(i int) float32 { return rowY(i) + (findRowH-findFieldH)/2 }
	x := g.bar.X + findPad
	g.r[hitChevron] = render.Rect{X: x, Y: g.bar.Y + 4, W: findChevW, H: g.bar.H - 8}
	x += findChevW + 2
	fieldX := x
	g.r[hitQuery] = render.Rect{X: fieldX, Y: fieldY(0), W: fieldW, H: findFieldH}
	tY := fieldY(0) + (findFieldH-20)/2
	right := fieldX + fieldW - 3
	for _, h := range []findHit{hitRegex, hitWord, hitCase} {
		right -= findToggleW
		g.r[h] = render.Rect{X: right, Y: tY, W: findToggleW, H: 20}
	}
	x = fieldX + fieldW + 6
	x += findCountW
	btnY := rowY(0) + (findRowH-findBtnW)/2
	for _, h := range []findHit{hitPrev, hitNext, hitInSel, hitClose} {
		g.r[h] = render.Rect{X: x, Y: btnY, W: findBtnW, H: findBtnW}
		x += findBtnW + 2
	}
	if e.search.replaceMode {
		g.r[hitReplace] = render.Rect{X: fieldX, Y: fieldY(1), W: fieldW, H: findFieldH}
		g.r[hitPreserve] = render.Rect{X: fieldX + fieldW - 3 - findToggleW, Y: fieldY(1) + (findFieldH-20)/2, W: findToggleW, H: 20}
		x = fieldX + fieldW + 6
		by := rowY(1) + (findRowH-findBtnW)/2
		for _, h := range []findHit{hitReplOne, hitReplAll} {
			g.r[h] = render.Rect{X: x, Y: by, W: findBtnW, H: findBtnW}
			x += findBtnW + 4
		}
	}
	return g
}

// hitAt names the part of the bar under (x, y).
func (g *findGeom) hitAt(x, y float32) findHit {
	if !g.bar.Contains(x, y) {
		return hitNone
	}
	// Toggles sit inside the inputs, so they are tested first.
	for _, h := range []findHit{hitCase, hitWord, hitRegex, hitPreserve, hitChevron, hitPrev, hitNext, hitInSel, hitClose, hitReplOne, hitReplAll, hitQuery, hitReplace} {
		if r := g.r[h]; r.W > 0 && r.Contains(x, y) {
			return h
		}
	}
	return hitBar
}

// findMouse handles the pointer over the bar and reports whether it took the
// event.
func (e *Editor) findMouse(m *input.Mouse) bool {
	s := &e.search
	if !s.open || s.query == nil {
		return false
	}
	g := e.findGeom()
	s.query.host.Frame = g.r[hitQuery]
	s.repl.host.Frame = g.r[hitReplace]
	// A drag that started in an input stays with it, past the bar's edge.
	for _, f := range []*TextInput{s.query, s.repl} {
		if f.dragging {
			f.onMouse(f.host, m)
			m.Consumed = true
			return true
		}
	}
	hit := g.hitAt(m.X, m.Y)
	if hit != s.hover {
		s.hover, s.hoverAt = hit, time.Now()
	}
	if !m.Pressed && !m.RightPressed {
		return false
	}
	if hit == hitNone {
		if m.Pressed {
			s.focused = false // a click in the text hands it the keyboard
			e.syncFindFocus()
		}
		return false
	}
	if m.Consumed {
		return true
	}
	s.focused = true
	switch hit {
	case hitQuery, hitReplace:
		f := s.query
		if hit == hitReplace {
			f = s.repl
		}
		e.focusFind(hit, false)
		f.onMouse(f.host, m)
	case hitBar:
	default:
		if m.Pressed {
			e.findActivate(hit)
		}
	}
	m.Consumed = true
	return true
}

// findTip is the tooltip of a bar button: what it does and its shortcut.
func (e *Editor) findTip(h findHit) string {
	tips := map[findHit]struct {
		title string
		act   EditorAction
	}{
		hitChevron:  {"Toggle Replace", ActionReplace},
		hitCase:     {"Match Case", ActionFindToggleCase},
		hitWord:     {"Match Whole Word", ActionFindToggleWord},
		hitRegex:    {"Use Regular Expression", ActionFindToggleRegex},
		hitPreserve: {"Preserve Case", ActionFindTogglePreserve},
		hitPrev:     {"Previous Match", ActionFindPrev},
		hitNext:     {"Next Match", ActionFindNext},
		hitInSel:    {"Find in Selection", ActionFindToggleInSel},
		hitClose:    {"Close", ActionFindClose},
		hitReplOne:  {"Replace", ActionReplaceOne},
		hitReplAll:  {"Replace All", ActionReplaceAll},
	}
	t, ok := tips[h]
	if !ok {
		return ""
	}
	if k := e.ShortcutLabel(t.act); k != "" {
		return t.title + " (" + k + ")"
	}
	return t.title
}

// findCountText is the match counter: "3 of 12", "No results", or a
// problem with the query.
func (e *Editor) findCountText() (string, bool) {
	s := &e.search
	switch {
	case s.badRegex:
		return "Invalid regex", true
	case s.query.Value == "":
		return "", false
	case len(s.matches) == 0 && s.pending:
		return "…", false
	case len(s.matches) == 0:
		return "No results", true
	}
	total := strconv.Itoa(len(s.matches))
	if s.capped {
		total += "+"
	}
	return strconv.Itoa(s.current+1) + " of " + total, false
}

func (e *Editor) paintFindBar(dl *render.DrawList, _ *shape.Engine) {
	s := &e.search
	if !s.open || s.query == nil {
		return
	}
	th := theme.Current()
	text := frameText()
	sheet := frameIcons()
	g := e.findGeom()
	radius := th.Radius.Large

	shadow := g.bar
	shadow.Y += 2
	dl.AddRoundedRect(shadow, radius+1, render.Color{A: 0.3})
	dl.AddRoundedRectBorder(g.bar, radius, 1, th.Chrome, th.Border)

	button := func(h findHit, icon icons.Icon, active, disabled bool) {
		r := g.r[h]
		col := th.ForegroundMuted
		switch {
		case disabled:
			col = th.ForegroundDisabled
		case active:
			dl.AddRoundedRect(r, th.Radius.Medium, alpha(th.Accent, 0.25))
			dl.AddRoundedRectStroke(r, render.UniformCorners(th.Radius.Medium), render.UniformBorder(1), alpha(th.Accent, 0.8), render.BorderSolid)
			col = th.Foreground
		case s.hover == h:
			dl.AddRoundedRect(r, th.Radius.Medium, alpha(th.Foreground, 0.1))
			col = th.Foreground
		}
		if s.focused && s.focus == h && e.focused {
			dl.AddRoundedRectStroke(r, render.UniformCorners(th.Radius.Medium), render.UniformBorder(1), th.FocusRing, render.BorderSolid)
		}
		sz := min(findIconSz, r.H-4)
		sheet.Draw(dl, icon, render.Rect{X: r.X + (r.W-sz)/2, Y: r.Y + (r.H-sz)/2, W: sz, H: sz}, col)
	}

	chev := icons.ChevronRight
	if s.replaceMode {
		chev = icons.ChevronDown
	}
	if !e.readOnly {
		button(hitChevron, chev, false, false)
	}

	countStr, bad := e.findCountText()
	fieldBg := th.Background
	if bad && s.query.Value != "" {
		fieldBg = th.ErrorSurface
	}
	q := s.query
	q.host.Frame = g.r[hitQuery]
	q.embed = fieldEmbed{on: true, padR: 3*findToggleW + 4, bg: fieldBg, line: th.Border}
	if bad && s.query.Value != "" {
		q.embed.fg, q.embed.hasFg = th.ErrorForeground, true
	}
	q.paint(dl, nil)
	button(hitCase, icons.CaseSensitive, s.caseSens, false)
	button(hitWord, icons.WholeWord, s.wholeWord, false)
	button(hitRegex, icons.Regex, s.regex, false)

	body := th.Typography.Body.Size
	if countStr != "" {
		_, ch := text.MeasureAt(countStr, body)
		cx := g.r[hitQuery].X + g.r[hitQuery].W + 6
		cy := g.r[hitQuery].Y + (findFieldH-ch)/2
		col := th.TextDim
		if bad {
			col = th.ErrorForeground
		}
		dl.PushClip(render.Rect{X: cx, Y: g.bar.Y, W: findCountW - 2, H: findRowH})
		text.DrawStringTopAt(dl, countStr, cx, cy, col, body)
		dl.PopClip()
	}
	none := len(s.matches) == 0
	button(hitPrev, icons.ArrowUp, false, none)
	button(hitNext, icons.ArrowDown, false, none)
	button(hitInSel, icons.TextAlignStart, s.inSel, false)
	button(hitClose, icons.X, false, false)

	if s.replaceMode {
		r := s.repl
		r.host.Frame = g.r[hitReplace]
		r.embed = fieldEmbed{on: true, padR: findToggleW + 4, bg: th.Background, line: th.Border}
		r.paint(dl, nil)
		button(hitPreserve, icons.CaseUpper, s.preserveCase, false)
		button(hitReplOne, icons.Replace, false, none || e.readOnly)
		button(hitReplAll, icons.ReplaceAll, false, none || e.readOnly)
	}

	if h := s.hover; h != hitNone && h != hitBar && time.Since(s.hoverAt) >= findTipDelay {
		if tip := e.findTip(h); tip != "" {
			e.paintFindTip(dl, g, g.r[h], tip)
		}
	}
}

// paintFindTip draws a tooltip under anchor, kept inside the viewport.
func (e *Editor) paintFindTip(dl *render.DrawList, g findGeom, anchor render.Rect, tip string) {
	th := theme.Current()
	text := frameText()
	size := th.Typography.Caption.Size
	tw, thh := text.MeasureAt(tip, size)
	box := render.Rect{W: tw + 12, H: thh + 8}
	box.X = anchor.X + anchor.W/2 - box.W/2
	box.Y = g.bar.Y + g.bar.H + 4
	vp := e.viewport.Frame
	box.X = clampf(box.X, vp.X+2, vp.X+vp.W-box.W-2)
	dl.AddRoundedRectBorder(box, th.Radius.Medium, 1, th.Chrome, th.Border)
	text.DrawStringTopAt(dl, tip, box.X+6, box.Y+4, th.Foreground, size)
}
