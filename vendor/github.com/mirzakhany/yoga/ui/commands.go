package ui

import (
	"strings"
	"unicode"

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

func rowHeightFor(cmd *Command, th *theme.Theme) float32 {
	if cmd != nil && cmd.section {
		return commandSectionHeight(th)
	}
	return commandRowHeight(th)
}

// CommandsHost is the window-owned command registry and searchable palette.
type CommandsHost struct {
	scrim  *Scrim
	panel  *layout.Element
	Open   bool
	query  string
	cursor int // index into filtered list

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
	h.cmds = h.cmds[:0]
	h.byID = make(map[string]int)
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

// ToggleLabel returns the display label for the toggle chord.
func (h *CommandsHost) ToggleLabel() string {
	return h.toggle.Label()
}

// Show opens the palette and clears the query.
func (h *CommandsHost) Show() {
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
	h.filtered = filterCommands(h.cmds, h.query)
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

// FilterKeys steals navigation/activate keys while the palette is open.
func (h *CommandsHost) FilterKeys(keys []input.KeyEvent) (pass []input.KeyEvent) {
	if !h.Open {
		return keys
	}
	h.rebuildFilter()
	moved := false
	for _, ev := range keys {
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
		rh := rowHeightFor(cmd, th)
		if i == index {
			return y, rh
		}
		y += rh
	}
	return y, commandRowHeight(th)
}

func (h *CommandsHost) contentHeight(th *theme.Theme) float32 {
	var hgt float32
	for _, cmd := range h.filtered {
		hgt += rowHeightFor(cmd, th)
	}
	return hgt
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
	rowH := commandRowHeight(th)
	// Keep panel height stable whether the list is full, short, or empty so the
	// search field does not jump as the query filters results.
	listH := float32(commandsListMaxRows) * rowH
	searchH := th.Metrics.ControlHeight + th.Spacing.M*2
	dh := searchH + listH + th.Spacing.S
	if dh > commandsPanelMaxH {
		dh = commandsPanelMaxH
	}
	dh = f32min(dh, vh-float32(th.Spacing.XXXL)*2)
	h.listView = f32max(0, dh-searchH)
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
	search := TextField("__commands-query", h.query).
		Placeholder("Type a command…").
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
			Subtitle("No commands"),
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

	return Column(
		Row(search).Padding(th.Spacing.M).Shrink(0),
		HLine(th.Stroke.Thin, th.Border),
		ViewOf(body).Grow(1),
	).Grow(1).Background(TokenChrome).
		Style(Spec{}.Radius(th.Radius.Large).Border(TokenBorder, th.Stroke.Thin))
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
	rowH := commandRowHeight(th)
	active := i == h.cursor
	hovered := i == h.hoverRow && !active

	kids := make([]View, 0, 4)
	if !cmd.icon.Empty() {
		kids = append(kids, Icon(cmd.icon, th.Metrics.IconSizeSM, th.ForegroundMuted))
	}
	titleKids := []View{Text(cmd.DisplayTitle())}
	if sub := cmd.subtitle(); sub != "" {
		titleKids = append(titleKids, Caption(sub))
	}
	kids = append(kids, Column(titleKids...).Gap(th.Spacing.XXS).Grow(1))
	if cmd.shortcut.Valid() {
		kids = append(kids, Kbd(cmd.shortcut.Label()))
	}

	row := Row(kids...).
		Gap(th.Spacing.M).
		PaddingXY(th.Spacing.M, th.Spacing.M).
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
				h.Hide()
				h.run(cmdCopy)
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
// Registration order is preserved so Section headers stay with their items.
// A Section is kept only when at least one following entry (until the next
// Section) is visible.
func filterCommands(cmds []*Command, q string) []*Command {
	q = strings.TrimSpace(q)
	out := make([]*Command, 0, len(cmds))
	for i, cmd := range cmds {
		if cmd == nil || cmd.hidden {
			continue
		}
		if cmd.section {
			if sectionHasVisible(cmds, i+1, q) {
				out = append(out, cmd)
			}
			continue
		}
		if q == "" {
			out = append(out, cmd)
			continue
		}
		if _, ok := matchScore(cmd.DisplayTitle(), cmd.id, cmd.group, cmd.detail, q); ok {
			out = append(out, cmd)
		}
	}
	return out
}

// sectionHasVisible reports whether any non-section entry after start (until
// the next section) should appear for query q.
func sectionHasVisible(cmds []*Command, start int, q string) bool {
	for i := start; i < len(cmds); i++ {
		cmd := cmds[i]
		if cmd == nil || cmd.hidden {
			continue
		}
		if cmd.section {
			return false
		}
		if q == "" {
			return true
		}
		if _, ok := matchScore(cmd.DisplayTitle(), cmd.id, cmd.group, cmd.detail, q); ok {
			return true
		}
	}
	return false
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
	h := []rune(strings.ToLower(hay))
	n := []rune(strings.ToLower(needle))
	if len(n) == 0 {
		return 0, true
	}
	hi := 0
	score := 0
	consec := 0
	first := -1
	for _, nr := range n {
		found := false
		for hi < len(h) {
			hr := h[hi]
			hi++
			if equalFoldRune(hr, nr) {
				if first < 0 {
					first = hi - 1
				}
				consec++
				score += 10 + consec*5
				found = true
				break
			}
			consec = 0
		}
		if !found {
			return 0, false
		}
	}
	if first == 0 {
		score += 50
	}
	score += max(0, 40-len(h))
	return score, true
}

func equalFoldRune(a, b rune) bool {
	if a == b {
		return true
	}
	return unicode.ToLower(a) == unicode.ToLower(b)
}