package ui

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const (
	commandsPanelWidth   = float32(560)
	commandsPanelMaxH    = float32(420)
	commandsPanelTopFrac = float32(0.12)
	commandsListMaxRows  = 10
	commandsListID       = "__commands-list"
)

// commandRowHeight is the uniform selectable list-row height (title + subtitle + padding).
func commandRowHeight(th *theme.Theme) float32 {
	padY := th.Spacing.M * 2
	return padY + th.Typography.Caption.LineHeight + th.Spacing.XXS + th.Typography.Body.LineHeight
}

// commandSectionHeight is the shorter labeled separator row.
func commandSectionHeight(th *theme.Theme) float32 {
	return th.Spacing.S + th.Typography.Caption.LineHeight + th.Spacing.S
}

// Density selects how tall list rows are.
type Density int

const (
	// DensityDefault keeps two-line palette rows: title over subtitle.
	DensityDefault Density = iota
	// DensityCompact uses single-line rows of Metrics.ControlHeight, with the
	// subtitle dropped and the trailing detail on the same line.
	DensityCompact
)

// rowHeight is the height of a selectable row at the host's density.
func (h *CommandsHost) rowHeight(th *theme.Theme) float32 {
	if h.density == DensityCompact {
		return th.Metrics.ControlHeight
	}
	return commandRowHeight(th)
}

func (h *CommandsHost) rowHeightFor(cmd *Command, th *theme.Theme) float32 {
	if cmd != nil && cmd.section {
		return commandSectionHeight(th)
	}
	return h.rowHeight(th)
}

// commandFooterHeight is the hint footer's height.
func commandFooterHeight(th *theme.Theme) float32 {
	return th.Spacing.S*2 + th.Typography.Caption.LineHeight + th.Stroke.Thin
}

// CommandsHost is the window-owned command registry and searchable palette.
type CommandsHost struct {
	scrim  *Scrim
	panel  *layout.Element
	Open   bool
	query  string
	cursor int // index into filtered list

	// scope limits the palette to entries registered with that Scope; empty
	// shows the unscoped entries. placeholder overrides the search hint.
	scope       string
	placeholder string

	cmds     []*Command // this frame's registrations
	byID     map[string]int
	toggle   Chord
	filtered []*Command
	hoverRow int // mouse highlight; -1 = none (persists across frames)
	ctx      *Ctx

	listSV   *ScrollView
	listView float32
	searchH  float32 // search chrome height; used to clear hover over the field

	lastPointerX, lastPointerY float32
	pointerSeen                bool

	density    Density
	hintFooter bool
}

var _ View = (*CommandsHost)(nil)
var _ Focusable = (*CommandsHost)(nil)
var _ KeyFilterer = (*CommandsHost)(nil)

// NewCommandsHost builds a closed command host with default toggle Mod+K.
func NewCommandsHost() *CommandsHost {
	return &CommandsHost{
		scrim:    NewScrim(),
		byID:     make(map[string]int),
		hoverRow: -1,
		toggle:   MustParseChord("Mod+K"),
	}
}

// beginFrame clears per-frame registrations. Open/query/cursor/hover persist so
// the paint rebuild can still show the highlight set during mouse dispatch.
func (h *CommandsHost) beginFrame() {
	// Commands close over application state, so release the references rather
	// than just resetting the length.
	clear(h.cmds)
	h.cmds = h.cmds[:0]
	h.byID = make(map[string]int)
	clear(h.filtered)
	h.filtered = h.filtered[:0]
	h.ctx = nil
}

// Register adds or replaces commands for this frame (last write wins per id).
func (h *CommandsHost) Register(cmds ...*Command) {
	for _, cmd := range cmds {
		if cmd == nil || cmd.id == "" {
			continue
		}
		if i, ok := h.byID[cmd.id]; ok {
			h.cmds[i] = cmd
			continue
		}
		h.byID[cmd.id] = len(h.cmds)
		h.cmds = append(h.cmds, cmd)
	}
}

// ToggleChord overrides the palette open/close shortcut (default Mod+K).
func (h *CommandsHost) ToggleChord(s string) *CommandsHost {
	if ch, err := ParseChord(s); err == nil {
		h.toggle = ch
	}
	return h
}

// Density sets the palette's row height: DensityCompact shows single-line
// rows so more results fit.
func (h *CommandsHost) Density(d Density) *CommandsHost {
	h.density = d
	return h
}

// HintFooter shows a footer listing the palette's keys: ↑↓ to navigate, ↵ to
// run, the highlighted entry's Alt chord, and esc to close.
func (h *CommandsHost) HintFooter(v bool) *CommandsHost {
	h.hintFooter = v
	return h
}

// ToggleLabel returns the display label for the toggle chord.
func (h *CommandsHost) ToggleLabel() string {
	return h.toggle.Label()
}

// Show opens the palette on the unscoped commands and clears the query.
func (h *CommandsHost) Show() { h.ShowScope("", "") }

// ShowScope opens the palette on the entries registered with Scope(scope),
// with placeholder as the search hint (a default when empty). Toggling or
// hiding the palette returns it to the unscoped commands.
func (h *CommandsHost) ShowScope(scope, placeholder string) {
	h.scope = scope
	h.placeholder = placeholder
	h.Open = true
	h.query = ""
	h.cursor = 0
	h.hoverRow = -1
	h.pointerSeen = false
	if h.listSV != nil {
		h.listSV.scrollY = 0
		h.listSV.syncScroll()
	}
}

// Hide closes the palette without running a command.
func (h *CommandsHost) Hide() {
	h.Open = false
	h.scope = ""
	h.placeholder = ""
	h.query = ""
	h.cursor = 0
	h.hoverRow = -1
	h.pointerSeen = false
	h.scrim.Hide()
}

// Toggle opens or closes the palette.
func (h *CommandsHost) Toggle() {
	if h.Open {
		h.Hide()
	} else {
		h.Show()
	}
}

// Dispatch consumes matching shortcut keys from kb. Call after mouse dispatch
// and before yoga.KeyHook / focus routing. Removes consumed events from kb.Keys.
func (h *CommandsHost) Dispatch(kb *input.Keyboard) {
	if kb == nil || len(kb.Keys) == 0 {
		return
	}
	if h.modalBlocked() {
		return
	}
	kept := kb.Keys[:0]
	for _, ev := range kb.Keys {
		if h.toggle.Valid() && h.toggle.Matches(ev) {
			h.Toggle()
			continue
		}
		if h.Open {
			// While open, skip other command shortcuts so typing does not fire them.
			kept = append(kept, ev)
			continue
		}
		if cmd := h.matchShortcut(ev); cmd != nil {
			h.run(cmd)
			continue
		}
		kept = append(kept, ev)
	}
	kb.Keys = kept
}

func (h *CommandsHost) modalBlocked() bool {
	c := h.ctx
	if c == nil {
		return false
	}
	if c.dialogs != nil && c.dialogs.Open {
		return true
	}
	if c.files != nil && c.files.Open {
		return true
	}
	return false
}

func (h *CommandsHost) matchShortcut(ev input.KeyEvent) *Command {
	for _, cmd := range h.cmds {
		if cmd == nil || cmd.section || !cmd.enabled || !cmd.shortcut.Valid() {
			continue
		}
		if cmd.shortcut.Matches(ev) {
			return cmd
		}
	}
	return nil
}

func (h *CommandsHost) run(cmd *Command) {
	if cmd == nil || cmd.section || !cmd.enabled || cmd.run == nil {
		return
	}
	cmd.run()
}

func (h *CommandsHost) rebuildFilter() {
	h.filtered = filterCommands(h.inScope(), h.query)
	if len(h.filtered) == 0 {
		h.cursor = 0
		return
	}
	if h.cursor < 0 {
		h.cursor = 0
	}
	if h.cursor >= len(h.filtered) {
		h.cursor = len(h.filtered) - 1
	}
	h.clampCursorToSelectable()
}

// inScope returns this frame's commands that belong to the open scope.
func (h *CommandsHost) inScope() []*Command {
	out := make([]*Command, 0, len(h.cmds))
	for _, cmd := range h.cmds {
		if cmd != nil && cmd.scope == h.scope {
			out = append(out, cmd)
		}
	}
	return out
}

// FilterKeys steals navigation/activate keys while the palette is open.
func (h *CommandsHost) FilterKeys(keys []input.KeyEvent) (pass []input.KeyEvent) {
	if !h.Open {
		return keys
	}
	h.rebuildFilter()
	moved := false
	for _, ev := range keys {
		if cmd := h.cursorCommand(); cmd != nil && cmd.altRun != nil && cmd.alt.Matches(ev) {
			h.activateAlt(cmd)
			continue
		}
		switch ev.Key {
		case input.KeyUp:
			h.moveCursor(-1)
			moved = true
		case input.KeyDown:
			h.moveCursor(1)
			moved = true
		case input.KeyHome:
			h.cursor = 0
			h.clampCursorToSelectable()
			moved = true
		case input.KeyEnd:
			if len(h.filtered) > 0 {
				h.cursor = len(h.filtered) - 1
				h.clampCursorToSelectable()
			}
			moved = true
		case input.KeyEnter:
			h.activateCursor()
		default:
			pass = append(pass, ev)
		}
	}
	if moved {
		h.ensureCursorVisible()
	}
	return pass
}

func (h *CommandsHost) moveCursor(delta int) {
	n := len(h.filtered)
	if n == 0 || delta == 0 {
		return
	}
	for range n {
		h.cursor += delta
		if h.cursor < 0 {
			h.cursor = n - 1
		} else if h.cursor >= n {
			h.cursor = 0
		}
		if !h.filtered[h.cursor].section {
			return
		}
	}
}

func (h *CommandsHost) clampCursorToSelectable() {
	n := len(h.filtered)
	if n == 0 {
		return
	}
	if h.cursor < 0 {
		h.cursor = 0
	}
	if h.cursor >= n {
		h.cursor = n - 1
	}
	if h.filtered[h.cursor].section {
		h.moveCursor(1)
		if h.filtered[h.cursor].section {
			// Only sections in the list.
			h.cursor = 0
		}
	}
}

func (h *CommandsHost) ensureCursorVisible() {
	sv := h.listSV
	if sv == nil || h.listView <= 0 || len(h.filtered) == 0 {
		return
	}
	th := theme.Current()
	top, rowH := h.rowOffset(h.cursor, th)
	contentH := h.contentHeight(th)
	sv.contentH = contentH
	if sv.vbar != nil && sv.vbar.ContentHeight != nil {
		*sv.vbar.ContentHeight = contentH
	}
	if sv.host != nil && sv.host.Frame.H < h.listView {
		sv.host.Frame.H = h.listView
	}
	bottom := top + rowH
	if top < sv.scrollY {
		sv.scrollY = top
	} else if bottom > sv.scrollY+h.listView {
		sv.scrollY = bottom - h.listView
	}
	sv.syncScroll()
}

func (h *CommandsHost) rowOffset(index int, th *theme.Theme) (top, height float32) {
	var y float32
	for i, cmd := range h.filtered {
		rh := h.rowHeightFor(cmd, th)
		if i == index {
			return y, rh
		}
		y += rh
	}
	return y, h.rowHeight(th)
}

func (h *CommandsHost) contentHeight(th *theme.Theme) float32 {
	var hgt float32
	for _, cmd := range h.filtered {
		hgt += h.rowHeightFor(cmd, th)
	}
	return hgt
}

// cursorCommand is the highlighted selectable entry, or nil.
func (h *CommandsHost) cursorCommand() *Command {
	if h.cursor < 0 || h.cursor >= len(h.filtered) {
		return nil
	}
	if cmd := h.filtered[h.cursor]; cmd != nil && !cmd.section {
		return cmd
	}
	return nil
}

// activateAlt closes the palette and runs cmd's Alt action.
func (h *CommandsHost) activateAlt(cmd *Command) {
	if cmd == nil || !cmd.enabled || cmd.altRun == nil {
		return
	}
	h.Hide()
	cmd.altRun()
}

func (h *CommandsHost) activateCursor() {
	h.rebuildFilter()
	if h.cursor < 0 || h.cursor >= len(h.filtered) {
		return
	}
	cmd := h.filtered[h.cursor]
	if cmd == nil || cmd.section || !cmd.enabled {
		return
	}
	h.Hide()
	h.run(cmd)
}

// Layout registers the scrim and palette panel while open.
func (h *CommandsHost) Layout(c *Ctx) *layout.Element {
	h.ctx = c
	if !h.Open {
		return layout.New(layout.Box().Size(0, 0))
	}
	h.rebuildFilter()
	vw, vh := c.Viewport()
	th := c.Theme()
	dw := f32min(commandsPanelWidth, vw-float32(th.Spacing.XXXL)*2)
	if dw < 280 {
		dw = f32max(280, vw-float32(th.Spacing.L)*2)
	}
	rowH := h.rowHeight(th)
	// Keep panel height stable whether the list is full, short, or empty so the
	// search field does not jump as the query filters results.
	listH := float32(commandsListMaxRows) * rowH
	searchH := th.Metrics.ControlHeight + th.Spacing.M*2
	footH := float32(0)
	if h.hintFooter {
		footH = commandFooterHeight(th)
	}
	dh := searchH + listH + th.Spacing.S + footH
	if dh > commandsPanelMaxH {
		dh = commandsPanelMaxH
	}
	dh = f32min(dh, vh-float32(th.Spacing.XXXL)*2)
	h.listView = f32max(0, dh-searchH-footH)
	h.searchH = searchH

	x := f32max(0, (vw-dw)/2)
	y := f32max(0, vh*commandsPanelTopFrac)
	if y+dh > vh-float32(th.Spacing.L) {
		y = f32max(0, vh-dh-float32(th.Spacing.L))
	}

	h.scrim.Show(0, 0, vw, vh)
	scrimHost := h.scrim.host
	scrimHost.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if h.Open && e.Frame.Contains(m.X, m.Y) {
			m.ScrollY = 0
			m.ScrollX = 0
			m.Consumed = true
			if h.panel == nil || !h.panel.Frame.Contains(m.X, m.Y) {
				h.hoverRow = -1
				if m.Pressed {
					h.Hide()
				}
			}
		}
	}
	scrimAt := len(c.overlays)
	c.Overlay(scrimHost)

	if c.Focus() != nil {
		c.Focus().BeginModal()
	}
	inner := h.chrome(c)
	var innerEl *layout.Element
	if inner != nil {
		innerEl = inner.Layout(c)
	}
	host := layout.New(layout.Box().Absolute(x, y).Size(dw, dh), innerEl)
	host.Overlay = true
	host.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		r := th.Radius.Large
		drawElevationShadow(dl, host.Frame, r, th.Elevation.ShadowLg)
	}
	host.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if !e.Frame.Contains(m.X, m.Y) {
			return
		}
		// Rows set hoverRow while the pointer is over them (children run first).
		// Clear it when the pointer is over the search chrome instead.
		if m.Y < e.Frame.Y+h.searchH {
			h.hoverRow = -1
		}
		m.ScrollY = 0
		m.ScrollX = 0
		m.Consumed = true
	}
	h.panel = host
	insert := scrimAt + 1
	if insert >= len(c.overlays) {
		c.Overlay(host)
	} else {
		c.overlays = append(c.overlays[:insert], append([]*layout.Element{host}, c.overlays[insert:]...)...)
	}
	if c.Focus() != nil {
		c.Focus().SetModal(h)
	}
	return layout.New(layout.Box().Size(0, 0))
}

func (h *CommandsHost) chrome(c *Ctx) View {
	th := c.Theme()
	placeholder := h.placeholder
	if placeholder == "" {
		placeholder = "Type a command…"
	}
	search := TextField("__commands-query", h.query).
		Placeholder(placeholder).
		IconStart(icons.Search).
		OnChange(func(s string) {
			h.query = s
			h.cursor = 0
			if h.listSV != nil {
				h.listSV.scrollY = 0
				h.listSV.syncScroll()
			}
		}).
		DefaultFocus().
		Grow(1)

	var body View
	if len(h.filtered) == 0 {
		h.listSV = nil
		// Compact empty message — avoid EmptyState's XXL padding, which fights
		// the fixed panel height and makes the search chrome feel like it moved.
		body = Center(Column(
			Icon(icons.Search, th.Metrics.IconSizeMD*2, th.ForegroundMuted),
			Subtitle(h.emptyTitle()),
			Muted("Try a different search"),
		).Gap(th.Spacing.S).Align(AlignCenter)).Grow(1)
	} else {
		rows := make([]View, 0, len(h.filtered))
		for i, cmd := range h.filtered {
			rows = append(rows, &commandRow{host: h, idx: i, cmd: cmd})
		}
		// Keep a handle for keyboard-driven scroll-into-view; do not call
		// ensureCursorVisible here — that would fight mouse-wheel scrolling
		// whenever the highlight is above the current offset (e.g. cursor 0).
		sv := c.Widget(commandsListID, func() any { return NewScrollView(nil) }).(*ScrollView)
		h.listSV = sv
		body = Scroll(commandsListID, Column(rows...)).Grow(1)
	}

	parts := []View{
		Row(search).Padding(th.Spacing.M).Shrink(0),
		HLine(th.Stroke.Thin, th.Border),
		ViewOf(body).Grow(1),
	}
	if h.hintFooter {
		parts = append(parts, HLine(th.Stroke.Thin, th.Border), h.footer(th))
	}
	return Column(parts...).Grow(1).Background(TokenChrome).
		Style(Spec{}.Radius(th.Radius.Large).Border(TokenBorder, th.Stroke.Thin))
}

// footer lists the palette's keys.
func (h *CommandsHost) footer(th *theme.Theme) View {
	hint := func(key, label string) View {
		return Row(Kbd(key), Caption(label).Style(Spec{}.TextColor(TokenForegroundMuted))).
			Gap(th.Spacing.XS).Align(AlignCenter)
	}
	kids := []View{hint("↑↓", "Navigate"), hint("↵", "Run")}
	if cmd := h.cursorCommand(); cmd != nil && cmd.altRun != nil {
		label := cmd.altLabel
		if label == "" {
			label = "Alternate"
		}
		kids = append(kids, hint(cmd.alt.Label(), label))
	}
	kids = append(kids, Spacer(), hint("esc", "Close"))
	return Row(kids...).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.S).
		Height(commandFooterHeight(th) - th.Stroke.Thin).Align(AlignCenter).Shrink(0)
}

func (h *CommandsHost) emptyTitle() string {
	if h.scope != "" {
		return "No matches"
	}
	return "No commands"
}

type commandRow struct {
	host *CommandsHost
	idx  int
	cmd  *Command
}

func (r *commandRow) Layout(c *Ctx) *layout.Element {
	h, i, cmd := r.host, r.idx, r.cmd
	th := c.Theme()
	if cmd.section {
		return r.layoutSection(c, th)
	}
	rowH := h.rowHeight(th)
	active := i == h.cursor
	hovered := i == h.hoverRow && !active
	compact := h.density == DensityCompact

	kids := make([]View, 0, 5)
	switch {
	case cmd.leading != nil:
		kids = append(kids, cmd.leading)
	case !cmd.icon.Empty():
		kids = append(kids, Icon(cmd.icon, th.Metrics.IconSizeSM, th.ForegroundMuted))
	}
	title := matchTitle(cmd.DisplayTitle(), h.query, active)
	if compact {
		kids = append(kids, ViewOf(title).Grow(1))
	} else {
		titleKids := []View{title}
		if sub := cmd.subtitle(); sub != "" {
			titleKids = append(titleKids, Caption(sub))
		}
		kids = append(kids, Column(titleKids...).Gap(th.Spacing.XXS).Grow(1))
	}
	if cmd.trailing != "" {
		kids = append(kids, Caption(cmd.trailing).Style(Spec{}.TextColor(TokenForegroundMuted)).Ellipsis(EllipsisEnd).Shrink(1))
	}
	if cmd.shortcut.Valid() {
		kids = append(kids, Kbd(cmd.shortcut.Label()))
	}

	padY := th.Spacing.M
	if compact {
		padY = 0
	}
	row := Row(kids...).
		Gap(th.Spacing.M).
		PaddingXY(th.Spacing.M, padY).
		Height(rowH).
		Align(AlignCenter)
	if active {
		row = row.Background(TokenListActive)
	} else if hovered {
		row = row.Background(TokenListHover)
	}
	if !cmd.enabled {
		row = row.Disabled(true)
	}
	el := row.Layout(c)
	idx := i
	cmdCopy := cmd
	prev := el.OnMouse
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if prev != nil {
			prev(e, m)
		}
		if e.Frame.Contains(m.X, m.Y) {
			h.hoverRow = idx
			moved := !h.pointerSeen || m.X != h.lastPointerX || m.Y != h.lastPointerY
			h.lastPointerX, h.lastPointerY = m.X, m.Y
			h.pointerSeen = true
			if moved {
				h.cursor = idx
			}
			m.SetCursor(input.CursorPointer)
			if m.Released && cmdCopy.enabled {
				h.cursor = idx
				if cmdCopy.altRun != nil && m.Mods.Primary() {
					h.activateAlt(cmdCopy)
				} else {
					h.Hide()
					h.run(cmdCopy)
				}
				m.Consumed = true
			}
		}
	}
	return el
}

func (r *commandRow) layoutSection(c *Ctx, th *theme.Theme) *layout.Element {
	h := commandSectionHeight(th)
	label := strings.ToUpper(r.cmd.DisplayTitle())
	row := Row(
		Caption(label).Style(Spec{}.TextColor(TokenForegroundMuted)),
		Spacer(),
	).PaddingXY(th.Spacing.M, th.Spacing.S).Height(h).Align(AlignCenter)
	el := row.Layout(c)
	// Sections are not interactive; clear hover if the pointer rests on one.
	idx := r.idx
	host := r.host
	prev := el.OnMouse
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if prev != nil {
			prev(e, m)
		}
		if e.Frame.Contains(m.X, m.Y) && host.hoverRow == idx {
			host.hoverRow = -1
		}
	}
	return el
}

func (h *CommandsHost) Focus()                   {}
func (h *CommandsHost) Blur()                    {}
func (h *CommandsHost) Focused() bool            { return h.Open }
func (h *CommandsHost) CapturesTab() bool        { return false }
func (h *CommandsHost) FocusOnClick() bool       { return false }
func (h *CommandsHost) FocusEl() *layout.Element { return h.panel }
func (h *CommandsHost) HandleText([]rune)        {}

func (h *CommandsHost) HandleKeys(keys []input.KeyEvent) {
	if !h.Open {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEscape {
			h.Hide()
			return
		}
	}
}

// filterCommands returns visible commands matching q.
// Sections keep their place and their items stay under them; within each
// section, matches are ranked best first, ties keeping registration order.
// With an empty query every visible entry is listed in registration order.
// A Section is kept only when at least one of its entries (until the next
// Section) is visible.
func filterCommands(cmds []*Command, q string) []*Command {
	q = strings.TrimSpace(q)
	out := make([]*Command, 0, len(cmds))
	type match struct {
		cmd   *Command
		score int
	}
	var header *Command
	var group []match
	flush := func() {
		if len(group) == 0 {
			return
		}
		if header != nil {
			out = append(out, header)
		}
		slices.SortStableFunc(group, func(a, b match) int { return b.score - a.score })
		for _, m := range group {
			out = append(out, m.cmd)
		}
		group = group[:0]
	}
	for _, cmd := range cmds {
		if cmd == nil || cmd.hidden {
			continue
		}
		if cmd.section {
			flush()
			header = cmd
			continue
		}
		if q == "" {
			group = append(group, match{cmd: cmd})
			continue
		}
		if score, ok := matchScore(cmd.DisplayTitle(), cmd.id, cmd.group, cmd.detail, q); ok {
			group = append(group, match{cmd: cmd, score: score})
		}
	}
	flush()
	return out
}

// matchScore is a case-insensitive subsequence match. Higher is better.
func matchScore(title, id, group, detail, q string) (int, bool) {
	best := -1
	for _, hay := range []string{title, id, group, detail} {
		if hay == "" {
			continue
		}
		if s, ok := subsequenceScore(hay, q); ok && s > best {
			best = s
		}
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

func subsequenceScore(hay, needle string) (int, bool) {
	score, _, ok := subsequenceMatch(hay, needle, false)
	return score, ok
}

// subsequenceMatch matches needle against hay as a case-insensitive
// subsequence, taking the leftmost occurrence of each rune. With keep it also
// returns the byte offset in hay of every matched rune.
func subsequenceMatch(hay, needle string, keep bool) (score int, pos []int, ok bool) {
	if needle == "" {
		return 0, nil, true
	}
	if keep {
		pos = make([]int, 0, len(needle))
	}
	runes := 0
	hi := 0
	consec := 0
	first := -1
	for _, nr := range needle {
		found := false
		for hi < len(hay) {
			hr, size := utf8.DecodeRuneInString(hay[hi:])
			at := hi
			hi += size
			runes++
			if equalFoldRune(hr, nr) {
				if first < 0 {
					first = runes - 1
				}
				consec++
				score += 10 + consec*5
				if keep {
					pos = append(pos, at)
				}
				found = true
				break
			}
			consec = 0
		}
		if !found {
			return 0, nil, false
		}
	}
	runes += utf8.RuneCountInString(hay[hi:])
	if first == 0 {
		score += 50
	}
	score += max(0, 40-runes)
	return score, pos, true
}

// FuzzyMatch reports whether query is a case-insensitive subsequence of s,
// the way the command palette matches, and returns the byte ranges of s that
// matched, with adjacent runes merged into one range. Use it to highlight
// matches in custom lists.
func FuzzyMatch(s, query string) (ranges [][2]int, ok bool) {
	query = strings.TrimSpace(query)
	_, pos, ok := subsequenceMatch(s, query, true)
	if !ok {
		return nil, false
	}
	for _, p := range pos {
		_, size := utf8.DecodeRuneInString(s[p:])
		if n := len(ranges); n > 0 && ranges[n-1][1] == p {
			ranges[n-1][1] = p + size
			continue
		}
		ranges = append(ranges, [2]int{p, p + size})
	}
	return ranges, true
}

// matchTitle is a row title with the runes matching q drawn in SemiBold, and
// in the link color unless the row is highlighted.
func matchTitle(s, q string, active bool) View {
	ranges, ok := FuzzyMatch(s, q)
	if !ok || len(ranges) == 0 {
		return Text(s).Ellipsis(EllipsisEnd)
	}
	return &matchText{text: s, ranges: ranges, active: active}
}

type matchText struct {
	text   string
	ranges [][2]int
	active bool
}

func (m *matchText) Layout(c *Ctx) *layout.Element {
	th := c.Theme()
	size := th.Typography.Body.Size
	fg := th.Foreground
	if c.env.hasColor {
		fg = c.env.textColor
	}
	hl := th.Link
	if m.active || c.env.hasColor {
		hl = fg
	}
	type seg struct {
		s      string
		weight int
		col    render.Color
		w      float32
	}
	var segs []seg
	add := func(s string, weight int, col render.Color) {
		if s == "" {
			return
		}
		w := size * 0.5 * float32(len(s))
		if eng := c.Text(); eng != nil {
			w, _ = eng.MeasureAtWeight(s, size, weight)
		}
		segs = append(segs, seg{s, weight, col, w})
	}
	at := 0
	for _, r := range m.ranges {
		add(m.text[at:r[0]], shape.WeightRegular, fg)
		add(m.text[r[0]:r[1]], shape.WeightSemiBold, hl)
		at = r[1]
	}
	add(m.text[at:], shape.WeightRegular, fg)
	var tw float32
	for _, sg := range segs {
		tw += sg.w
	}
	lh := th.Typography.Body.LineHeight
	if eng := c.Text(); eng != nil {
		_, lh = eng.MeasureAtWeight(m.text, size, shape.WeightRegular)
	}
	el := layout.New(layout.Box().Size(tw, lh).FlexShrink(1).Min(0, lh))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		dl.PushClip(el.Frame)
		x := el.Frame.X
		for _, sg := range segs {
			text.DrawStringTopAtWeight(dl, sg.s, x, el.Frame.Y, sg.col, size, sg.weight)
			x += sg.w
		}
		dl.PopClip()
	}
	return el
}

func equalFoldRune(a, b rune) bool {
	if a == b {
		return true
	}
	return unicode.ToLower(a) == unicode.ToLower(b)
}
