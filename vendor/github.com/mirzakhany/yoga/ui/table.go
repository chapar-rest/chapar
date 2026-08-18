package ui

import (
	"sort"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TableColumnKind describes how a column is rendered and interacted with.
type TableColumnKind int

const (
	TableColText TableColumnKind = iota
	TableColEditable
	TableColCheckbox
	TableColActions
)

// TableColumn defines one column in a Table.
type TableColumn struct {
	ID    string
	Label string
	Kind  TableColumnKind
	Width float32 // fixed px; 0 = flex remainder shared equally among flex columns
	// Sortable enables click-to-sort on the header label (text/editable columns).
	Sortable bool
	// Locked fixes the column: no header sort and no resize handle on its right edge.
	Locked bool
}

// TableAction is an icon button in a TableColActions column.
type TableAction struct {
	Icon    string
	Tooltip string
	OnClick func(rowID string)
}

// TableRow is one data row. Cells map column IDs to display/edit values.
type TableRow struct {
	ID       string
	Cells    map[string]string
	Selected bool
	// Icon is drawn to the left of the first TableColText cell when set.
	Icon string
}

// Table is a scrollable, self-painted table with optional filter, row selection,
// inline cell editing, and per-row actions.
type Table struct {
	host    *layout.Element
	Columns []TableColumn
	Rows    []TableRow
	Actions []TableAction

	Filter            string
	OnCellChange      func(rowID, colID, value string)
	OnSelectionChange func()
	OnDelete          func(rowID string)
	OnSortChange      func(colID string, asc bool)
	OnRowClick        func(rowID string)
	OnRowActivate     func(rowID string)

	// Selectable enables click-to-select on the row body (not just checkboxes).
	Selectable bool
	// MultiSelect allows Cmd/Ctrl toggle and Shift range selection.
	MultiSelect bool

	vbar *Scrollbar

	scrollY          float32
	contentH         float32
	visible          []int // indices into Rows after filter
	rowH, headerH    float32
	hoverRow         int
	hoverAction      int
	headerCheckHover bool
	editingRowID     string
	editingColID     string
	editField        *TextInput
	editOriginal     string
	focused          bool

	colWidths      []float32
	sortColID      string
	sortAsc        bool
	hoverResizeCol int
	resizeDragging bool
	resizeCol      int
	resizeStartX   float32
	resizeStartW   float32

	anchorRowID  string
	lastClickRow string
	lastClickAt  time.Time
}

const (
	tableMinColW       float32 = 48
	tableResizeHandleW float32 = 6
	tableDoubleClick           = 400 * time.Millisecond
)

// NewTable builds an empty table with the given columns and row actions.
func NewTable(columns []TableColumn, actions []TableAction) *Table {
	th := theme.Current()
	t := &Table{
		Columns:        columns,
		Actions:        actions,
		hoverRow:       -1,
		hoverResizeCol: -1,
		rowH:           th.Typography.Body.LineHeight + th.Spacing.S,
		headerH:        th.Typography.Body.LineHeight + th.Spacing.S,
	}
	if len(t.Actions) == 0 {
		t.Actions = []TableAction{{Icon: "close", Tooltip: "Delete"}}
	}

	chipH := t.rowH - th.Spacing.XS
	t.editField = NewTextInput(TextFieldConfig{
		Height:      chipH,
		Radius:      th.Radius.Small,
		BorderWidth: th.Stroke.Thin,
	})
	t.editField.host.Overlay = true

	barSize := th.Metrics.ScrollbarSize
	t.vbar = NewScrollbarAxis(Vertical, &t.scrollY, &t.contentH, barSize)

	t.host = layout.New(layout.Box().FlexGrow(1).H(200).Min(0, 200).FlexShrink(0), t.vbar.host)
	t.host.Clip = true
	t.host.Paint = t.paint
	t.host.OnMouse = t.onMouse

	t.colWidths = make([]float32, len(columns))
	for i, c := range columns {
		t.colWidths[i] = c.Width
	}

	t.rebuildVisible()
	return t
}

func (t *Table) Layout(c *Ctx) *layout.Element {
	if c.Focus() != nil {
		c.Focus().Add(t)
	}
	t.Update(c.Mouse())
	if t.editingRowID != "" && t.editField != nil && t.editField.host != nil {
		c.Overlay(t.editField.host)
	}
	return t.host
}

// SetRows replaces all rows and clears edit state.
func (t *Table) SetRows(rows []TableRow) {
	t.Rows = append([]TableRow(nil), rows...)
	t.cancelEdit()
	t.rebuildVisible()
	t.clampScroll()
}

// AddRow appends a row and returns its index in Rows.
func (t *Table) AddRow(row TableRow) int {
	if row.Cells == nil {
		row.Cells = map[string]string{}
	}
	t.Rows = append(t.Rows, row)
	t.rebuildVisible()
	return len(t.Rows) - 1
}

// RemoveRow removes a row by ID. Returns whether a row was removed.
func (t *Table) RemoveRow(id string) bool {
	for i, row := range t.Rows {
		if row.ID == id {
			t.Rows = append(t.Rows[:i], t.Rows[i+1:]...)
			if t.editingRowID == id {
				t.cancelEdit()
			}
			t.rebuildVisible()
			t.clampScroll()
			return true
		}
	}
	return false
}

// SetFilter restricts visible rows to those whose text/editable cells match query
// (case-insensitive substring). Empty query shows all rows.
func (t *Table) SetFilter(query string) {
	t.Filter = query
	t.rebuildVisible()
	t.clampScroll()
}

// SelectedIDs returns IDs of all selected rows.
func (t *Table) SelectedIDs() []string {
	var out []string
	for _, row := range t.Rows {
		if row.Selected {
			out = append(out, row.ID)
		}
	}
	return out
}

// SortBy sorts visible rows by the given column. Repeated calls toggle direction.
func (t *Table) SortBy(colID string) {
	i := t.colIndex(colID)
	if i < 0 || !t.colSortable(t.Columns[i]) {
		return
	}
	if t.editingRowID != "" {
		t.commitEdit()
	}
	if t.sortColID == colID {
		t.sortAsc = !t.sortAsc
	} else {
		t.sortColID = colID
		t.sortAsc = true
	}
	t.rebuildVisible()
	if t.OnSortChange != nil {
		t.OnSortChange(colID, t.sortAsc)
	}
}

// SortColumn returns the active sort column ID, or "" if unsorted.
func (t *Table) SortColumn() string { return t.sortColID }

// SortAscending reports whether the active sort is ascending.
func (t *Table) SortAscending() bool { return t.sortAsc }

// SetColumnWidth sets an explicit pixel width for a column (0 restores flex).
func (t *Table) SetColumnWidth(colID string, w float32) {
	i := t.colIndex(colID)
	if i < 0 || t.Columns[i].Locked {
		return
	}
	if w > 0 && w < tableMinColW {
		w = tableMinColW
	}
	t.colWidths[i] = w
}

// ColumnWidth returns the resolved width for a column at the current viewport size.
func (t *Table) ColumnWidth(colID string) float32 {
	i := t.colIndex(colID)
	if i < 0 {
		return 0
	}
	cw, _, _ := t.bodyMetrics()
	widths, _ := t.columnLayout(cw)
	return widths[i]
}

// SelectAll sets the selection state of all currently visible rows.
func (t *Table) SelectAll(checked bool) {
	for _, i := range t.visible {
		t.Rows[i].Selected = checked
	}
	if t.OnSelectionChange != nil {
		t.OnSelectionChange()
	}
}

func (t *Table) rebuildVisible() {
	t.visible = t.visible[:0]
	for i := range t.Rows {
		if t.rowMatches(i) {
			t.visible = append(t.visible, i)
		}
	}
	t.sortVisible()
	t.contentH = float32(len(t.visible)) * t.rowH
}

func (t *Table) sortVisible() {
	if t.sortColID == "" || len(t.visible) < 2 {
		return
	}
	sort.SliceStable(t.visible, func(i, j int) bool {
		a := strings.ToLower(t.Rows[t.visible[i]].Cells[t.sortColID])
		b := strings.ToLower(t.Rows[t.visible[j]].Cells[t.sortColID])
		if t.sortAsc {
			return a < b
		}
		return a > b
	})
}

func (t *Table) rowMatches(rowIdx int) bool {
	if t.Filter == "" {
		return true
	}
	q := strings.ToLower(t.Filter)
	row := t.Rows[rowIdx]
	for _, col := range t.Columns {
		if col.Kind == TableColText || col.Kind == TableColEditable {
			if strings.Contains(strings.ToLower(row.Cells[col.ID]), q) {
				return true
			}
		}
	}
	return false
}

func (t *Table) rowByID(id string) (int, bool) {
	for i, row := range t.Rows {
		if row.ID == id {
			return i, true
		}
	}
	return -1, false
}

func (t *Table) barSize() float32 {
	return theme.Current().Metrics.ScrollbarSize
}

func (t *Table) bodyMetrics() (clientW, clientH float32, vShow bool) {
	f := t.host.Frame
	clientW, clientH = f.W, f.H-t.headerH
	vShow = t.contentH > clientH
	if vShow {
		clientW = f.W - t.barSize()
	}
	return clientW, clientH, vShow
}

func (t *Table) bodyViewport() render.Rect {
	f := t.host.Frame
	cw, ch, _ := t.bodyMetrics()
	return render.Rect{X: f.X, Y: f.Y + t.headerH, W: cw, H: ch}
}

func (t *Table) syncScrollbarLayout(vShow bool) {
	if vShow {
		t.vbar.host.Style = layout.Box().W(t.barSize()).AbsTop(t.headerH).AbsRight(0).AbsBottom(0)
		t.vbar.host.ReapplyStyle()
	}
}

func (t *Table) clampScroll() {
	_, clientH, _ := t.bodyMetrics()
	t.scrollY = clampf(t.scrollY, 0, f32max(0, t.contentH-clientH))
}

func (t *Table) colSortable(col TableColumn) bool {
	if col.Locked {
		return false
	}
	return col.Sortable
}

func (t *Table) colResizable(colIdx int) bool {
	if colIdx < 0 || colIdx >= len(t.Columns)-1 {
		return false
	}
	col := t.Columns[colIdx]
	if col.Locked {
		return false
	}
	if colIdx+1 < len(t.Columns) && t.Columns[colIdx+1].Locked {
		return false
	}
	return col.Kind == TableColText || col.Kind == TableColEditable
}

func (t *Table) columnLayout(viewportW float32) (widths, offsets []float32) {
	widths = make([]float32, len(t.Columns))
	flexN := 0
	fixed := float32(0)
	for i := range t.Columns {
		if t.colWidths[i] > 0 {
			widths[i] = t.colWidths[i]
			fixed += t.colWidths[i]
		} else {
			flexN++
		}
	}
	flexW := float32(0)
	if flexN > 0 {
		remain := viewportW - fixed
		if remain < 0 {
			remain = 0
		}
		flexW = remain / float32(flexN)
	}
	for i := range t.Columns {
		if t.colWidths[i] == 0 {
			widths[i] = flexW
		}
	}
	offsets = make([]float32, len(t.Columns))
	var x float32
	for i, w := range widths {
		offsets[i] = x
		x += w
	}
	return widths, offsets
}

func (t *Table) cellRect(rowY float32, colIdx int, widths, offsets []float32) render.Rect {
	f := t.host.Frame
	return render.Rect{
		X: f.X + offsets[colIdx],
		Y: rowY,
		W: widths[colIdx],
		H: t.rowH,
	}
}

func (t *Table) paintCheckbox(dl *render.DrawList, r render.Rect, checked, hovered bool) {
	th := theme.Current()
	box := th.Metrics.IconSizeSM
	bx := r.X + (r.W-box)/2
	by := r.Y + (r.H-box)/2
	br := render.Rect{X: bx, Y: by, W: box, H: box}
	border := th.Border
	fill := th.Chrome
	if checked {
		fill = th.Accent
		border = th.Accent
	}
	if hovered {
		fill = th.ListHover
		if checked {
			fill = th.AccentHover
		}
	}
	dl.AddRoundedRectBorder(br, th.Radius.Small, th.Stroke.Thin, fill, border)
	if checked {
		inner := render.Rect{X: bx + 2, Y: by + 2, W: box - 4, H: box - 4}
		frameIcons().Draw(dl, "check", inner, th.AccentForeground)
	}
}

func (t *Table) resizeHandleRect(cr render.Rect) render.Rect {
	return render.Rect{
		X: cr.X + cr.W - tableResizeHandleW/2,
		Y: cr.Y,
		W: tableResizeHandleW,
		H: cr.H,
	}
}

func (t *Table) paintHeader(dl *render.DrawList, text *shape.Engine, widths, offsets []float32) {
	th := theme.Current()
	f := t.host.Frame
	hdr := render.Rect{X: f.X, Y: f.Y, W: f.W, H: t.headerH}
	dl.AddRect(hdr, th.ChromeMuted)
	dl.PushClip(hdr)
	style := th.Typography.BodyStrong
	iconSz := th.Metrics.IconSizeSM
	for i, col := range t.Columns {
		cr := t.cellRect(f.Y, i, widths, offsets)
		switch col.Kind {
		case TableColCheckbox:
			t.paintCheckbox(dl, cr, t.allVisibleSelected(), t.headerCheckHover)
		default:
			tx := cr.X + t.padX()
			if col.Label != "" {
				lw, lh := text.MeasureAt(col.Label, style.Size)
				text.DrawStringTopAt(dl, col.Label, tx, cr.Y+(t.rowH-lh)/2, th.Foreground, style.Size)
				tx += lw + th.Spacing.XS
			}
			if t.colSortable(col) && t.sortColID == col.ID {
				icon := "expand_more"
				if t.sortAsc {
					icon = "expand_less"
				}
				ir := render.Rect{X: tx, Y: cr.Y + (t.rowH-iconSz)/2, W: iconSz, H: iconSz}
				frameIcons().Draw(dl, icon, ir, th.Accent)
			}
		}
		if t.colResizable(i) {
			hr := t.resizeHandleRect(cr)
			active := i == t.hoverResizeCol || (t.resizeDragging && i == t.resizeCol)
			lineCol := th.Border
			if active {
				lineCol = th.Accent
			}
			line := render.Rect{X: hr.X + (hr.W-1)/2, Y: hr.Y + th.Spacing.XS, W: 1, H: hr.H - 2*th.Spacing.XS}
			dl.AddRect(line, lineCol)
		}
	}
	dl.PopClip()
	dl.AddRect(render.Rect{X: f.X, Y: f.Y + t.headerH - th.Stroke.Thin, W: f.W, H: th.Stroke.Thin}, th.Border)
}

func (t *Table) allVisibleSelected() bool {
	if len(t.visible) == 0 {
		return false
	}
	for _, i := range t.visible {
		if !t.Rows[i].Selected {
			return false
		}
	}
	return true
}

func (t *Table) padX() float32 { return theme.Current().Spacing.S }

func (t *Table) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := t.host.Frame
	dl.AddRect(f, th.Chrome)

	cw, _, _ := t.bodyMetrics()
	widths, offsets := t.columnLayout(cw)
	t.paintHeader(dl, text, widths, offsets)

	vp := t.bodyViewport()
	dl.PushClip(vp)

	first := int(t.scrollY / t.rowH)
	if first < 0 {
		first = 0
	}
	for vi := first; vi < len(t.visible); vi++ {
		rowIdx := t.visible[vi]
		row := t.Rows[rowIdx]
		y := f.Y + t.headerH + float32(vi)*t.rowH - t.scrollY
		if y >= vp.Y+vp.H {
			break
		}

		if row.Selected {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListActive)
		} else if vi == t.hoverRow {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListHover)
		}

		for i, col := range t.Columns {
			cr := t.cellRect(y, i, widths, offsets)
			switch col.Kind {
			case TableColCheckbox:
				t.paintCheckbox(dl, cr, row.Selected, false)
			case TableColText, TableColEditable:
				val := row.Cells[col.ID]
				if t.editingRowID == row.ID && t.editingColID == col.ID {
					continue
				}
				style := th.Typography.Body
				_, lh := text.MeasureAt(val, style.Size)
				tx := cr.X + t.padX()
				if row.Icon != "" && i == t.firstTextCol() {
					if sheet := frameIcons(); sheet != nil {
						sz := th.Metrics.IconSizeSM
						ir := render.Rect{X: tx, Y: cr.Y + (t.rowH-sz)/2, W: sz, H: sz}
						sheet.Draw(dl, row.Icon, ir, th.ForegroundMuted)
						tx += sz + t.padX()
					}
				}
				clipR := render.Rect{X: tx, Y: cr.Y, W: f32max(0, cr.X+cr.W-tx-t.padX()), H: cr.H}
				dl.PushClip(clipR)
				text.DrawStringTopAt(dl, val, tx, cr.Y+(t.rowH-lh)/2, th.Foreground, style.Size)
				dl.PopClip()
			case TableColActions:
				iconSz, slot := t.actionSlotSize()
				pad := th.Spacing.XS
				for ai, act := range t.Actions {
					ax := t.actionSlotX(cr, ai)
					ir := render.Rect{X: ax + pad, Y: cr.Y + (t.rowH-iconSz)/2, W: iconSz, H: iconSz}
					actionCol := th.ForegroundMuted
					if vi == t.hoverRow && ai == t.hoverAction {
						bg := render.Rect{X: ax, Y: cr.Y + (t.rowH-slot)/2, W: slot, H: slot}
						dl.AddRoundedRect(bg, th.Radius.Small, th.ListHover)
						actionCol = th.Foreground
					}
					frameIcons().Draw(dl, act.Icon, ir, actionCol)
				}
			}
		}
	}

	dl.PopClip()

	if t.editingRowID != "" {
		t.positionEditField(widths, offsets)
	}
}

func (t *Table) positionEditField(widths, offsets []float32) {
	th := theme.Current()
	colIdx := t.colIndex(t.editingColID)
	if colIdx < 0 {
		return
	}
	f := t.host.Frame
	vi := t.visibleIndex(t.editingRowID)
	if vi < 0 {
		t.hideEditField()
		return
	}
	y := f.Y + t.headerH + float32(vi)*t.rowH - t.scrollY
	vp := t.bodyViewport()
	if y < vp.Y || y+t.rowH > vp.Y+vp.H {
		t.hideEditField()
		return
	}
	cr := t.cellRect(y, colIdx, widths, offsets)
	pad := th.Spacing.XS
	t.editField.host.Frame = render.Rect{
		X: cr.X + pad,
		Y: cr.Y + pad,
		W: cr.W - 2*pad,
		H: cr.H - 2*pad,
	}
}

func (t *Table) hideEditField() {
	t.editField.host.Frame = render.Rect{X: -10000, Y: -10000, W: 0, H: 0}
}

func (t *Table) colIndex(id string) int {
	for i, c := range t.Columns {
		if c.ID == id {
			return i
		}
	}
	return -1
}

func (t *Table) firstTextCol() int {
	for i, c := range t.Columns {
		if c.Kind == TableColText {
			return i
		}
	}
	return -1
}

func (t *Table) visibleIndex(rowID string) int {
	for vi, rowIdx := range t.visible {
		if t.Rows[rowIdx].ID == rowID {
			return vi
		}
	}
	return -1
}

func (t *Table) actionSlotSize() (iconSz, slot float32) {
	th := theme.Current()
	iconSz = th.Metrics.IconSizeSM
	slot = iconSz + 2*th.Spacing.XS
	return iconSz, slot
}

func (t *Table) actionSlotX(cr render.Rect, actionIdx int) float32 {
	_, slot := t.actionSlotSize()
	ax := cr.X + float32(actionIdx)*slot
	total := float32(len(t.Actions)) * slot
	if total < cr.W {
		ax = cr.X + float32(actionIdx)*slot + (cr.W-total)/2
	}
	return ax
}

func (t *Table) overScrollbar(m *input.Mouse) bool {
	_, _, vShow := t.bodyMetrics()
	if vShow && t.vbar.host.Frame.Contains(m.X, m.Y) {
		return true
	}
	return false
}

func (t *Table) startColumnResize(colIdx int, x float32, currentW float32) {
	t.resizeDragging = true
	t.resizeCol = colIdx
	t.resizeStartX = x
	t.resizeStartW = currentW
	if t.colWidths[colIdx] > 0 {
		t.resizeStartW = t.colWidths[colIdx]
	}
	t.colWidths[colIdx] = t.resizeStartW
}

func (t *Table) updateColumnResize(m *input.Mouse) {
	delta := m.X - t.resizeStartX
	t.colWidths[t.resizeCol] = f32max(tableMinColW, t.resizeStartW+delta)
}

func (t *Table) handleHeaderMouse(el *layout.Element, m *input.Mouse, widths, offsets []float32) {
	t.headerCheckHover = false
	t.hoverResizeCol = -1

	for i := range t.Columns {
		if !t.colResizable(i) {
			continue
		}
		cr := t.cellRect(el.Frame.Y, i, widths, offsets)
		if t.resizeHandleRect(cr).Contains(m.X, m.Y) {
			t.hoverResizeCol = i
			m.SetCursor(input.CursorResizeEW)
			if m.Pressed {
				t.startColumnResize(i, m.X, widths[i])
				m.Consumed = true
			}
			return
		}
	}

	for i, col := range t.Columns {
		cr := t.cellRect(el.Frame.Y, i, widths, offsets)
		switch col.Kind {
		case TableColCheckbox:
			if cr.Contains(m.X, m.Y) {
				t.headerCheckHover = true
				if m.Released {
					t.SelectAll(!t.allVisibleSelected())
					m.Consumed = true
				}
				if m.Pressed {
					m.Consumed = true
				}
			}
		default:
			if t.colSortable(col) && cr.Contains(m.X, m.Y) && m.Released {
				t.SortBy(col.ID)
				m.Consumed = true
			}
			if t.colSortable(col) && cr.Contains(m.X, m.Y) && m.Pressed {
				m.Consumed = true
			}
		}
	}
}

func (t *Table) onMouse(el *layout.Element, m *input.Mouse) {
	t.hoverRow = -1
	t.hoverAction = -1
	if !t.resizeDragging {
		t.hoverResizeCol = -1
	}
	t.headerCheckHover = false

	if t.resizeDragging {
		m.SetCursor(input.CursorResizeEW)
		if m.Down {
			t.updateColumnResize(m)
			m.Consumed = true
		}
		return
	}

	if el.Frame.Contains(m.X, m.Y) && (m.ScrollY != 0 || m.ScrollX != 0) {
		t.vbar.ApplyWheel(m, el.Frame)
		t.clampScroll()
	}

	if !el.Frame.Contains(m.X, m.Y) || t.overScrollbar(m) {
		return
	}

	cw, _, _ := t.bodyMetrics()
	widths, offsets := t.columnLayout(cw)

	if m.Y < el.Frame.Y+t.headerH {
		t.handleHeaderMouse(el, m, widths, offsets)
		return
	}

	vi := int((m.Y - el.Frame.Y - t.headerH + t.scrollY) / t.rowH)
	if vi < 0 || vi >= len(t.visible) {
		if m.Released && t.editingRowID != "" {
			t.commitEdit()
		}
		return
	}
	t.hoverRow = vi
	rowIdx := t.visible[vi]
	row := &t.Rows[rowIdx]
	y := el.Frame.Y + t.headerH + float32(vi)*t.rowH - t.scrollY
	handled := false

	for i, col := range t.Columns {
		cr := t.cellRect(y, i, widths, offsets)
		if !cr.Contains(m.X, m.Y) {
			continue
		}
		switch col.Kind {
		case TableColCheckbox:
			if m.Released {
				row.Selected = !row.Selected
				if t.OnSelectionChange != nil {
					t.OnSelectionChange()
				}
				m.Consumed = true
			}
			if m.Pressed {
				m.Consumed = true
			}
			handled = true
		case TableColEditable:
			if m.Released {
				if t.editingRowID != "" && (t.editingRowID != row.ID || t.editingColID != col.ID) {
					t.commitEdit()
				}
				t.startEdit(row.ID, col.ID)
				m.Consumed = true
			}
			if m.Pressed {
				m.Consumed = true
			}
			handled = true
		case TableColActions:
			_, slot := t.actionSlotSize()
			for ai, act := range t.Actions {
				ax := t.actionSlotX(cr, ai)
				ar := render.Rect{X: ax, Y: cr.Y + (t.rowH-slot)/2, W: slot, H: slot}
				if ar.Contains(m.X, m.Y) {
					t.hoverAction = ai
					if m.Released {
						t.fireAction(act, row.ID)
						m.Consumed = true
					}
					if m.Pressed {
						m.Consumed = true
					}
					handled = true
				}
			}
		}
	}

	if !handled && t.Selectable {
		if m.Pressed {
			m.Consumed = true
		}
		if m.Released {
			t.applyRowClick(row.ID, vi, m.Mods)
			if t.OnRowClick != nil {
				t.OnRowClick(row.ID)
			}
			now := time.Now()
			if t.lastClickRow == row.ID && now.Sub(t.lastClickAt) <= tableDoubleClick {
				t.lastClickRow = ""
				if t.OnRowActivate != nil {
					t.OnRowActivate(row.ID)
				}
			} else {
				t.lastClickRow = row.ID
				t.lastClickAt = now
			}
			m.Consumed = true
		}
	}
}

func (t *Table) applyRowClick(rowID string, visibleIdx int, mods input.Mod) {
	switch {
	case t.MultiSelect && mods.Has(input.ModShift) && t.anchorRowID != "":
		a := t.visibleIndex(t.anchorRowID)
		b := visibleIdx
		if a < 0 {
			a = b
		}
		lo, hi := a, b
		if lo > hi {
			lo, hi = hi, lo
		}
		for i := range t.Rows {
			t.Rows[i].Selected = false
		}
		for vi := lo; vi <= hi && vi < len(t.visible); vi++ {
			t.Rows[t.visible[vi]].Selected = true
		}
	case t.MultiSelect && mods.Primary():
		if idx, ok := t.rowByID(rowID); ok {
			t.Rows[idx].Selected = !t.Rows[idx].Selected
		}
		t.anchorRowID = rowID
	default:
		for i := range t.Rows {
			t.Rows[i].Selected = t.Rows[i].ID == rowID
		}
		t.anchorRowID = rowID
	}
	if t.OnSelectionChange != nil {
		t.OnSelectionChange()
	}
}

func (t *Table) fireAction(act TableAction, rowID string) {
	if act.OnClick != nil {
		act.OnClick(rowID)
		return
	}
	if t.RemoveRow(rowID) && t.OnDelete != nil {
		t.OnDelete(rowID)
	}
}

func (t *Table) startEdit(rowID, colID string) {
	rowIdx, ok := t.rowByID(rowID)
	if !ok {
		return
	}
	val := t.Rows[rowIdx].Cells[colID]
	t.editingRowID = rowID
	t.editingColID = colID
	t.editOriginal = val
	t.editField.setValue(val)
	t.editField.Focus()
}

func (t *Table) commitEdit() {
	if t.editingRowID == "" {
		return
	}
	rowIdx, ok := t.rowByID(t.editingRowID)
	if !ok {
		t.cancelEdit()
		return
	}
	val := t.editField.Value
	if t.Rows[rowIdx].Cells == nil {
		t.Rows[rowIdx].Cells = map[string]string{}
	}
	t.Rows[rowIdx].Cells[t.editingColID] = val
	rowID, colID := t.editingRowID, t.editingColID
	t.cancelEdit()
	if val != t.editOriginal && t.OnCellChange != nil {
		t.OnCellChange(rowID, colID, val)
	}
}

func (t *Table) cancelEdit() {
	t.editingRowID = ""
	t.editingColID = ""
	t.editOriginal = ""
	t.editField.Blur()
	t.hideEditField()
}

// Update drives the scrollbar and inline edit overlay. Call once per frame after layout.
func (t *Table) Update(m *input.Mouse) {
	if m == nil {
		_, _, vShow := t.bodyMetrics()
		t.syncScrollbarLayout(vShow)
		t.clampScroll()
		return
	}
	if t.resizeDragging {
		m.SetCursor(input.CursorResizeEW)
		if m.Down {
			t.updateColumnResize(m)
		} else {
			t.resizeDragging = false
			t.hoverResizeCol = -1
		}
	}
	_, _, vShow := t.bodyMetrics()
	t.syncScrollbarLayout(vShow)
	vp := t.bodyViewport()
	t.vbar.Update(m, vp)
	t.clampScroll()
	if t.editingRowID != "" {
		cw, _, _ := t.bodyMetrics()
		widths, offsets := t.columnLayout(cw)
		t.positionEditField(widths, offsets)
		t.editField.Update(m)
	}
}

func (t *Table) Focus() {
	t.focused = true
	if t.editingRowID != "" {
		t.editField.Focus()
	}
}

func (t *Table) Blur() {
	t.focused = false
	if t.editingRowID != "" {
		t.commitEdit()
	}
	t.editField.Blur()
}

func (t *Table) Focused() bool {
	return t.focused || t.editField.Focused()
}

func (t *Table) CapturesTab() bool { return false }

func (t *Table) FocusOnClick() bool { return true }

func (t *Table) FocusEl() *layout.Element { return t.host }

func (t *Table) HandleText(runes []rune) {
	if t.editingRowID != "" {
		t.editField.HandleText(runes)
	}
}

func (t *Table) HandleKeys(keys []input.KeyEvent) {
	if t.editingRowID != "" {
		for _, ev := range keys {
			if ev.Key == input.KeyEnter && ev.Mods == 0 {
				t.commitEdit()
				return
			}
		}
		t.editField.HandleKeys(keys)
		return
	}
	if !t.Selectable || t.OnRowActivate == nil {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEnter && ev.Mods == 0 {
			ids := t.SelectedIDs()
			if len(ids) > 0 {
				t.OnRowActivate(ids[0])
			}
			return
		}
	}
}

// CommitCellEdit commits the active inline edit (for tests).
func (t *Table) CommitCellEdit() { t.commitEdit() }

// StartCellEdit begins editing a cell programmatically (for tests).
func (t *Table) StartCellEdit(rowID, colID string) { t.startEdit(rowID, colID) }
