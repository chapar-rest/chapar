package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// FileDialogMode chooses whether the picker confirms files or folders.
type FileDialogMode int

const (
	FileDialogOpenFile FileDialogMode = iota
	FileDialogOpenFolder
)

// FileFilter restricts listed files by extension. Empty Exts means all files.
type FileFilter struct {
	Label string
	Exts  []string
}

// FileDialogOpts configures one Show of a FileDialog.
type FileDialogOpts struct {
	Title      string
	Mode       FileDialogMode
	Multiple   bool
	Dir        string
	Filters    []FileFilter
	ShowHidden bool
	OnConfirm  func(paths []string)
	OnCancel   func()
}

// FileDialog is a retained modal file/folder picker. Construct once with
// NewFileDialog, place it in the Body tree every frame, and call Show.
type FileDialog struct {
	Open bool

	scrim *Scrim
	table *Table
	panel *layout.Element

	opts       FileDialogOpts
	dir        string
	query      string
	searchOpen bool
	filterIdx  int
	showRecent bool
	entries    []fileEntry
	places     []filePlace
	recent     []string
	needFocus  bool
}

var _ View = (*FileDialog)(nil)
var _ Focusable = (*FileDialog)(nil)

const (
	fileDialogWidth  float32 = 720
	fileDialogHeight float32 = 480
	fileDialogMinW   float32 = 480
	fileDialogMinH   float32 = 320
)

// NewFileDialog builds a closed file picker host. Place it in the view tree so
// Layout can self-register the scrim and panel as overlays while open.
func NewFileDialog() *FileDialog {
	d := &FileDialog{scrim: NewScrim()}
	d.table = NewTable([]TableColumn{
		{ID: "name", Label: "Name", Kind: TableColText, Width: 0, Sortable: true},
		{ID: "size", Label: "Size", Kind: TableColText, Width: 88, Sortable: true},
		{ID: "type", Label: "Type", Kind: TableColText, Width: 88, Sortable: true},
		{ID: "modified", Label: "Modified", Kind: TableColText, Width: 110, Sortable: true},
	}, nil)
	d.table.Selectable = true
	d.table.OnRowActivate = d.activateRow
	st := d.table.host.Style
	st.Height = float32(math.NaN())
	st.MinHeight = 0
	st.Grow = 1
	st.Shrink = 1
	d.table.host.Style = st
	return d
}

// Show opens the picker with the given options.
func (d *FileDialog) Show(opts FileDialogOpts) {
	d.opts = opts
	d.query = ""
	d.searchOpen = false
	d.filterIdx = 0
	d.showRecent = false
	d.needFocus = true
	dir := opts.Dir
	if dir == "" {
		dir = defaultFileDialogDir()
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	d.dir = dir
	d.table.MultiSelect = opts.Multiple
	d.table.SetFilter("")
	d.Open = true
	d.reload()
}

// Close hides the picker without invoking OnCancel.
func (d *FileDialog) Close() {
	d.Open = false
	d.scrim.Hide()
}

// Layout registers the scrim and panel while open.
func (d *FileDialog) Layout(c *Ctx) *layout.Element {
	if !d.Open {
		return layout.New(layout.Box().Size(0, 0))
	}
	vw, vh := c.Viewport()
	th := c.Theme()
	margin := float32(th.Spacing.XXXL) * 2
	dw := f32min(fileDialogWidth, vw-margin)
	dh := f32min(fileDialogHeight, vh-margin)
	if dw < fileDialogMinW {
		dw = f32max(320, vw-margin)
	}
	if dh < fileDialogMinH {
		dh = f32max(240, vh-margin)
	}
	x := f32max(0, (vw-dw)/2)
	y := f32max(0, (vh-dh)/2)

	d.scrim.Show(0, 0, vw, vh)
	scrimAt := len(c.overlays)
	c.Overlay(d.scrim.host)

	if c.Focus() != nil {
		c.Focus().BeginModal()
	}
	inner := d.body(c).Layout(c)
	host := layout.New(layout.Box().Absolute(x, y).Size(dw, dh), inner)
	host.Overlay = true
	host.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		r := th.Radius.Large
		drawElevationShadow(dl, host.Frame, r, th.Elevation.ShadowLg)
	}
	host.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if e.Frame.Contains(m.X, m.Y) {
			m.ScrollY = 0
			m.ScrollX = 0
			m.Consumed = true
		}
	}
	d.panel = host
	// Place the panel immediately after this dialog's scrim. Other widgets
	// (tables, editors) may already have registered overlays; inserting at
	// index 1 would push the scrim on top of the panel.
	insert := scrimAt + 1
	if insert >= len(c.overlays) {
		c.Overlay(host)
	} else {
		c.overlays = append(c.overlays[:insert], append([]*layout.Element{host}, c.overlays[insert:]...)...)
	}
	if c.Focus() != nil {
		c.Focus().SetModal(d)
		if d.needFocus {
			c.Focus().Focus(d.table)
			d.needFocus = false
		}
	}
	return layout.New(layout.Box().Size(0, 0))
}

func (d *FileDialog) Focus()                   {}
func (d *FileDialog) Blur()                    {}
func (d *FileDialog) Focused() bool            { return d.Open }
func (d *FileDialog) CapturesTab() bool        { return false }
func (d *FileDialog) FocusOnClick() bool       { return false }
func (d *FileDialog) FocusEl() *layout.Element { return d.panel }
func (d *FileDialog) HandleText([]rune)        {}

func (d *FileDialog) HandleKeys(keys []input.KeyEvent) {
	if !d.Open {
		return
	}
	for _, ev := range keys {
		switch ev.Key {
		case input.KeyEscape:
			d.cancel()
			return
		case input.KeyEnter:
			d.confirm()
			return
		}
	}
}

func (d *FileDialog) body(c *Ctx) View {
	th := c.Theme()
	d.places = listFilePlaces(d.recent)
	return Column(
		d.header(th),
		HLine(th.Stroke.Thin, th.Border),
		Row(
			d.sidebar(th),
			VLine(th.Stroke.Thin, th.Border),
			d.mainPane(th),
		).Align(AlignStretch).Grow(1),
		d.footer(th),
	).Grow(1).Background(TokenChrome).Style(Spec{}.Radius(th.Radius.Large).Border(TokenBorder, th.Stroke.Thin))
}

func (d *FileDialog) header(th *theme.Theme) View {
	title := d.opts.Title
	if title == "" {
		title = d.defaultTitle()
	}
	openLabel := "Open"
	if d.opts.Mode == FileDialogOpenFolder {
		openLabel = "Select"
	}
	return Row(
		Button("fd-cancel", Text("Cancel")).OnClick(func() { d.cancel() }),
		Spacer(),
		Text(title).Style(Spec{}.TextColor(TokenForeground)),
		Spacer(),
		Row(
			IconButton("fd-search-toggle", "search").OnClick(func() {
				d.searchOpen = !d.searchOpen
				if !d.searchOpen {
					d.query = ""
					d.table.SetFilter("")
				}
			}),
			Button("fd-open", Text(openLabel)).Primary().Disabled(!d.canConfirm()).OnClick(func() { d.confirm() }),
		).Gap(th.Spacing.S),
	).Padding(th.Spacing.M).Gap(th.Spacing.M)
}

func (d *FileDialog) sidebar(th *theme.Theme) View {
	items := make([]NavItem, 0, len(d.places))
	for _, p := range d.places {
		items = append(items, NavItem{ID: p.ID, Label: p.Label, Icon: p.Icon})
	}
	if len(items) == 0 {
		items = []NavItem{{ID: "home", Label: "Home", Icon: "home"}}
	}
	bg := th.ChromeMuted
	return Nav("fd-places", NavVertical, NavIconLeft, items...).
		Selected(d.selectedPlace()).
		OnSelectItem(func(i int, id string) {
			if i < 0 || i >= len(d.places) {
				return
			}
			p := d.places[i]
			if p.ID == "recent" || id == "recent" {
				d.showRecent = true
				d.reload()
				return
			}
			if p.Path != "" {
				d.setDir(p.Path)
			}
		}).
		Width(180).
		NavBackground(&bg)
}

func (d *FileDialog) mainPane(th *theme.Theme) View {
	top := d.breadcrumb(th)
	if d.searchOpen {
		top = TextField("fd-search", d.query).
			Placeholder("Filter current folder…").
			IconStart("search").
			OnChange(func(s string) {
				d.query = s
				d.table.SetFilter(s)
			}).
			OnSubmit(func(string) { d.confirm() }).
			DefaultFocus().
			Grow(1)
	}
	return Column(
		top,
		ViewOf(d.table).Grow(1),
	).Grow(1).Background(TokenSurface).Padding(th.Spacing.XS)
}

func (d *FileDialog) breadcrumb(_ *theme.Theme) View {
	if d.showRecent {
		return Breadcrumb("fd-crumbs", BreadcrumbSegment{Label: "Recent"}).
			Height(theme.Current().Metrics.ControlHeight).PaddingXY(theme.Current().Spacing.S, 0)
	}
	th := theme.Current()
	segs := pathCrumbs(d.dir)
	crumbs := make([]BreadcrumbSegment, 0, len(segs))
	for _, p := range segs {
		path := p
		crumbs = append(crumbs, BreadcrumbSegment{
			Label: crumbLabel(path),
			OnSelect: func() {
				d.setDir(path)
			},
		})
	}
	return Breadcrumb("fd-crumbs", crumbs...).
		Height(th.Metrics.ControlHeight).PaddingXY(th.Spacing.S, 0)
}

func (d *FileDialog) footer(_ *theme.Theme) View {
	if d.opts.Mode != FileDialogOpenFile || len(d.opts.Filters) == 0 {
		return nil
	}
	opts := make([]SelectOption, 0, len(d.opts.Filters))
	for i, f := range d.opts.Filters {
		label := f.Label
		if label == "" {
			label = "Filter"
		}
		opts = append(opts, SelectOption{Label: label, Value: fmt.Sprintf("%d", i)})
	}
	sel := d.filterIdx
	if sel < 0 || sel >= len(opts) {
		sel = 0
	}
	th := theme.Current()
	return Row(
		Spacer(),
		Select("fd-filter", opts).Width(200).Selected(sel).OnChange(func(v string) {
			var i int
			if _, err := fmt.Sscanf(v, "%d", &i); err == nil && i >= 0 && i < len(d.opts.Filters) {
				d.filterIdx = i
				d.applyRows()
			}
		}),
	).Padding(th.Spacing.M)
}

func (d *FileDialog) defaultTitle() string {
	switch {
	case d.opts.Mode == FileDialogOpenFolder && d.opts.Multiple:
		return "Select Folders"
	case d.opts.Mode == FileDialogOpenFolder:
		return "Select Folder"
	case d.opts.Multiple:
		return "Open Files"
	default:
		return "Open File"
	}
}

func (d *FileDialog) selectedPlace() int {
	if d.showRecent {
		for i, p := range d.places {
			if p.ID == "recent" {
				return i
			}
		}
	}
	for i, p := range d.places {
		if p.Path != "" && sameFilePath(p.Path, d.dir) {
			return i
		}
	}
	return -1
}

func (d *FileDialog) setDir(path string) {
	d.showRecent = false
	d.dir = path
	d.reload()
}

func (d *FileDialog) reload() {
	if d.showRecent {
		d.entries = d.recentEntries()
	} else {
		d.entries = readFileEntries(d.dir, d.opts.ShowHidden)
	}
	d.applyRows()
}

func (d *FileDialog) recentEntries() []fileEntry {
	out := make([]fileEntry, 0, len(d.recent))
	for _, p := range d.recent {
		if e, ok := statFileEntry(p); ok {
			if d.opts.ShowHidden || len(e.Name) == 0 || e.Name[0] != '.' {
				out = append(out, e)
			}
		}
	}
	return out
}

func (d *FileDialog) currentFilter() FileFilter {
	if d.opts.Mode != FileDialogOpenFile || d.filterIdx < 0 || d.filterIdx >= len(d.opts.Filters) {
		return FileFilter{}
	}
	return d.opts.Filters[d.filterIdx]
}

func (d *FileDialog) applyRows() {
	list := d.entries
	if d.opts.Mode == FileDialogOpenFile {
		list = filterFileEntries(list, d.currentFilter())
	}
	rows := make([]TableRow, 0, len(list))
	for _, e := range list {
		icon := "file"
		if e.IsDir {
			icon = "folder"
		}
		rows = append(rows, TableRow{
			ID:   e.Path,
			Icon: icon,
			Cells: map[string]string{
				"name":     e.Name,
				"size":     formatFileSize(e.Size, e.IsDir),
				"type":     formatFileType(e.Name, e.IsDir),
				"modified": formatFileMod(e.ModTime),
			},
		})
	}
	d.table.SetRows(rows)
	d.table.SetFilter(d.query)
}

func (d *FileDialog) entryByPath(path string) (fileEntry, bool) {
	for _, e := range d.entries {
		if e.Path == path {
			return e, true
		}
	}
	return statFileEntry(path)
}

func (d *FileDialog) selectedEntries() []fileEntry {
	ids := d.table.SelectedIDs()
	out := make([]fileEntry, 0, len(ids))
	for _, id := range ids {
		if e, ok := d.entryByPath(id); ok {
			out = append(out, e)
		}
	}
	return out
}

func (d *FileDialog) canConfirm() bool {
	sel := d.selectedEntries()
	switch d.opts.Mode {
	case FileDialogOpenFolder:
		return true
	default:
		for _, e := range sel {
			if !e.IsDir {
				return true
			}
		}
		return len(sel) == 1 && sel[0].IsDir
	}
}

func (d *FileDialog) activateRow(id string) {
	e, ok := d.entryByPath(id)
	if !ok {
		return
	}
	if e.IsDir {
		d.setDir(e.Path)
		return
	}
	if d.opts.Mode == FileDialogOpenFile {
		d.finish([]string{e.Path})
	}
}

func (d *FileDialog) confirm() {
	if !d.Open {
		return
	}
	sel := d.selectedEntries()
	switch d.opts.Mode {
	case FileDialogOpenFolder:
		var paths []string
		for _, e := range sel {
			if e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		if len(paths) == 0 {
			paths = []string{d.dir}
		}
		if !d.opts.Multiple && len(paths) > 1 {
			paths = paths[:1]
		}
		d.finish(paths)
	default:
		var files, dirs []string
		for _, e := range sel {
			if e.IsDir {
				dirs = append(dirs, e.Path)
			} else {
				files = append(files, e.Path)
			}
		}
		if len(files) > 0 {
			if !d.opts.Multiple && len(files) > 1 {
				files = files[:1]
			}
			d.finish(files)
			return
		}
		if len(dirs) == 1 {
			d.setDir(dirs[0])
		}
	}
}

func (d *FileDialog) cancel() {
	if !d.Open {
		return
	}
	d.Close()
	if d.opts.OnCancel != nil {
		d.opts.OnCancel()
	}
}

func (d *FileDialog) finish(paths []string) {
	d.recent = rememberRecent(d.recent, paths)
	d.Close()
	if d.opts.OnConfirm != nil {
		d.opts.OnConfirm(paths)
	}
}
