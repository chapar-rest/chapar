// Package cookieui is the dialog for viewing and editing environment cookie jars.
package cookieui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/container"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// expiryLayouts are the formats accepted in the Expires field, in local time
// unless the value carries a zone.
var expiryLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

const expiryDisplay = "2006-01-02 15:04"

var sameSiteOptions = []ui.SelectOption{
	{Label: "Not set", Value: ""},
	{Label: "Lax", Value: "Lax"},
	{Label: "Strict", Value: "Strict"},
	{Label: "None", Value: "None"},
}

// Deps is what the dialog needs from the app.
type Deps struct {
	Store     *cookies.Store
	Envs      func() []*domain.Environment
	ActiveEnv func() *domain.Environment
	Error     func(error)
	Toast     func(string)
}

// Dialog lists and edits the cookies of one environment's jar.
type Dialog struct {
	deps Deps

	envID   string
	jar     *cookies.Jar
	version uint64 // jar version the table was built from
	table   *ui.Table
	query   string
	domain  string // "" shows every domain

	edit    *draft
	confirm clearKind
	clip    func() input.Clipboard
}

type draft struct {
	cookie  domain.Cookie
	isNew   bool
	expires string
	err     string
}

type clearKind int

const (
	clearNone clearKind = iota
	clearAll
	clearDomain
)

func New(deps Deps) *Dialog {
	d := &Dialog{deps: deps}
	d.table = ui.NewTable([]ui.TableColumn{
		// Value, Domain and Flags share the width left over, so none of them
		// collapses when the editor panel narrows the table.
		{ID: "name", Label: "Name", Kind: ui.TableColText, Width: 130, Sortable: true},
		{ID: "value", Label: "Value", Kind: ui.TableColText},
		{ID: "domain", Label: "Domain", Kind: ui.TableColText, Sortable: true},
		{ID: "path", Label: "Path", Kind: ui.TableColText, Width: 60, Sortable: true},
		{ID: "expires", Label: "Expires", Kind: ui.TableColText, Width: 80},
		{ID: "flags", Label: "Flags", Kind: ui.TableColText},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 64, Locked: true},
	}, []ui.TableAction{
		{Icon: icons.Copy, Tooltip: "Copy value"},
		{Icon: icons.Trash2, Tooltip: "Delete"},
	})
	d.table.Editable = false
	d.table.Selectable = true
	d.table.MinHeight = 120
	d.table.Actions[0].OnClick = d.copyValue
	d.table.Actions[1].OnClick = d.deleteCookie
	d.table.OnRowClick = d.selectCookie
	return d
}

// Show opens the dialog on the active environment's jar.
func (d *Dialog) Show(c *ui.Ctx) {
	d.envID = ""
	if env := d.deps.ActiveEnv(); env != nil {
		d.envID = env.ID()
	}
	d.query, d.domain = "", ""
	d.edit, d.confirm = nil, clearNone
	d.load()
	c.Dialogs().Show(ui.DialogOpts{
		Title:  "Cookies",
		Width:  1000,
		Height: 640,
		Body:   d.Layout,
		Actions: []ui.DialogAction{
			{Label: "Close", Primary: true},
		},
	})
}

func (d *Dialog) load() {
	d.jar = nil
	if d.deps.Store == nil {
		return
	}
	jar, err := d.deps.Store.For(d.envID)
	if err != nil {
		d.deps.Error(err)
		return
	}
	d.jar = jar
	d.refresh()
}

func (d *Dialog) all() []domain.Cookie {
	if d.jar == nil {
		return nil
	}
	return d.jar.All()
}

// refresh rebuilds the table rows from the jar with the current filters.
func (d *Dialog) refresh() {
	now := time.Now()
	q := strings.ToLower(strings.TrimSpace(d.query))
	var rows []ui.TableRow
	if d.jar != nil {
		d.version = d.jar.Version()
	}
	for _, c := range d.all() {
		if d.domain != "" && c.Domain != d.domain {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Name+"\x00"+c.Value+"\x00"+c.Domain), q) {
			continue
		}
		rows = append(rows, ui.TableRow{
			ID:       c.ID,
			Selected: d.edit != nil && d.edit.cookie.ID == c.ID,
			Cells: map[string]string{
				"name":    c.Name,
				"value":   c.Value,
				"domain":  displayDomain(c),
				"path":    c.Path,
				"expires": container.CookieExpiry(c, now),
				"flags":   container.CookieFlags(c),
			},
		})
	}
	d.table.SetRows(rows)
}

// displayDomain marks cookies sent to subdomains with a leading dot, as
// browser devtools do; host-only cookies show the bare host.
func displayDomain(c domain.Cookie) string {
	if c.HostOnly {
		return c.Domain
	}
	return "." + c.Domain
}

func (d *Dialog) save() {
	if d.deps.Store == nil {
		return
	}
	if err := d.deps.Store.Save(d.envID); err != nil {
		d.deps.Error(err)
	}
}

func (d *Dialog) find(id string) (domain.Cookie, bool) {
	for _, c := range d.all() {
		if c.ID == id {
			return c, true
		}
	}
	return domain.Cookie{}, false
}

func (d *Dialog) selectCookie(id string) {
	c, ok := d.find(id)
	if !ok {
		return
	}
	exp := ""
	if !c.Expires.IsZero() {
		exp = c.Expires.Local().Format(expiryDisplay)
	}
	d.edit = &draft{cookie: c, expires: exp}
	d.confirm = clearNone
}

func (d *Dialog) add() {
	d.edit = &draft{
		cookie: domain.Cookie{Domain: d.domain, Path: "/", Enabled: true, Source: domain.CookieSourceManual},
		isNew:  true,
	}
	d.confirm = clearNone
	d.refresh()
}

func (d *Dialog) copyValue(id string) {
	c, ok := d.find(id)
	if !ok || d.clip == nil {
		return
	}
	if clip := d.clip(); clip != nil {
		clip.Set(c.Value)
		d.deps.Toast("Copied")
	}
}

func (d *Dialog) deleteCookie(id string) {
	if d.jar == nil {
		return
	}
	d.jar.Delete(id)
	if d.edit != nil && d.edit.cookie.ID == id {
		d.edit = nil
	}
	d.save()
	d.refresh()
}

func (d *Dialog) commitEdit() {
	e := d.edit
	if e == nil || d.jar == nil {
		return
	}
	exp, err := parseExpiry(e.expires)
	if err != nil {
		e.err = err.Error()
		return
	}
	e.cookie.Expires = exp
	saved, err := d.jar.Upsert(e.cookie)
	if err != nil {
		e.err = err.Error()
		return
	}
	d.save()
	d.selectCookie(saved.ID)
	d.refresh()
	d.deps.Toast("Cookie saved")
}

func parseExpiry(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "session") {
		return time.Time{}, nil
	}
	for _, layout := range expiryLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("expires must look like %s, or be empty for a session cookie", expiryDisplay)
}

func (d *Dialog) clear(kind clearKind) {
	if d.jar == nil {
		return
	}
	switch kind {
	case clearAll:
		d.jar.ClearAll()
	case clearDomain:
		d.jar.ClearDomain(d.domain)
		d.domain = ""
	}
	d.confirm = clearNone
	d.edit = nil
	d.save()
	d.refresh()
}

func (d *Dialog) clearNow(fn func(*cookies.Jar)) func() {
	return func() {
		if d.jar == nil {
			return
		}
		fn(d.jar)
		if d.edit != nil && !d.edit.isNew {
			if _, ok := d.find(d.edit.cookie.ID); !ok {
				d.edit = nil
			}
		}
		d.save()
		d.refresh()
	}
}

// Layout is the dialog body.
func (d *Dialog) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	d.clip = c.Clipboard

	if d.deps.Store == nil {
		return ui.EmptyState("Cookies are not available", "This build does not keep a cookie jar.").EmptyIcon(icons.Cookie)
	}

	// Requests sent while the dialog is open change the jar.
	if d.jar != nil && d.jar.Version() != d.version {
		d.refresh()
	}

	rows := []ui.View{d.toolbar(th)}
	if d.confirm != clearNone {
		rows = append(rows, d.confirmRow(th))
	}

	var list ui.View = ui.ViewOf(d.table).Grow(1)
	if len(d.all()) == 0 {
		list = ui.Column(
			ui.EmptyState("No cookies in this jar",
				"Cookies the servers set are stored here per environment. You can also add one yourself.").
				EmptyIcon(icons.Cookie).
				Action(ui.Button("ck-empty-add", ui.Text("Add cookie")).IconStart(icons.Plus).OnClick(d.add)),
		).Grow(1)
	} else if len(d.table.Rows) == 0 {
		list = ui.Column(ui.EmptyState("No matches", "No cookie matches the filter.")).Grow(1)
	}

	main := []ui.View{ui.Column(list).Grow(1)}
	if d.edit != nil {
		main = append(main, d.editor(th))
	}
	rows = append(rows,
		ui.Row(main...).Gap(th.Spacing.M).Align(ui.AlignStretch).Grow(1),
		ui.Caption("Cookies are saved in the workspace's .state folder, which git ignores.").Shrink(0),
	)
	return ui.Column(rows...).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.XS).Grow(1)
}

func (d *Dialog) toolbar(th *theme.Theme) ui.View {
	envOpts := []ui.SelectOption{{Label: "No Environment", Value: ""}}
	envSel := 0
	for i, e := range d.deps.Envs() {
		envOpts = append(envOpts, ui.SelectOption{Label: e.MetaData.Name, Value: e.ID()})
		if e.ID() == d.envID {
			envSel = i + 1
		}
	}

	counts := map[string]int{}
	for _, c := range d.all() {
		counts[c.Domain]++
	}
	domains := make([]string, 0, len(counts))
	for dm := range counts {
		domains = append(domains, dm)
	}
	sort.Strings(domains)
	domOpts := []ui.SelectOption{{Label: fmt.Sprintf("All domains (%d)", len(d.all())), Value: ""}}
	domSel := 0
	for i, dm := range domains {
		domOpts = append(domOpts, ui.SelectOption{Label: fmt.Sprintf("%s (%d)", dm, counts[dm]), Value: dm})
		if dm == d.domain {
			domSel = i + 1
		}
	}

	clearItems := []ui.MenuItem{
		{Label: "Expired cookies", OnSelect: d.clearNow((*cookies.Jar).ClearExpired)},
		{Label: "Session cookies", OnSelect: d.clearNow((*cookies.Jar).ClearSession)},
	}
	if d.domain != "" {
		clearItems = append(clearItems, ui.MenuItem{Label: "All cookies of " + d.domain + "…", OnSelect: func() { d.confirm = clearDomain }})
	}
	clearItems = append(clearItems,
		ui.MenuSeparator,
		ui.MenuItem{Label: "All cookies…", Disabled: len(d.all()) == 0, OnSelect: func() { d.confirm = clearAll }},
	)

	return ui.Row(
		ui.Select("ck-env", envOpts).Width(180).Selected(envSel).OnChange(func(v string) {
			d.envID, d.domain = v, ""
			d.edit, d.confirm = nil, clearNone
			d.load()
		}),
		ui.Select("ck-domain", domOpts).Width(220).Selected(domSel).OnChange(func(v string) {
			d.domain = v
			d.confirm = clearNone
			d.refresh()
		}),
		ui.TextField("ck-search", d.query).
			Placeholder("Filter by name, value or domain").
			IconStart(icons.Search).
			OnChange(func(s string) {
				d.query = s
				d.refresh()
			}).Grow(1),
		ui.MenuButton("ck-clear", "Clear", clearItems),
		ui.Button("ck-add", ui.Text("Add")).IconStart(icons.Plus).OnClick(d.add),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).Shrink(0)
}

func (d *Dialog) confirmRow(th *theme.Theme) ui.View {
	n := len(d.all())
	msg := fmt.Sprintf("Delete all %d cookies in this jar? This cannot be undone.", n)
	if d.confirm == clearDomain {
		n = 0
		for _, c := range d.all() {
			if c.Domain == d.domain {
				n++
			}
		}
		msg = fmt.Sprintf("Delete %d cookies of %s? This cannot be undone.", n, d.domain)
	}
	kind := d.confirm
	return ui.Row(
		ui.Icon(icons.TriangleAlert, th.Metrics.IconSizeMD, th.Warning),
		ui.Text(msg).Grow(1),
		ui.Button("ck-confirm-cancel", ui.Text("Cancel")).OnClick(func() { d.confirm = clearNone }),
		ui.Button("ck-confirm-delete", ui.Text("Delete")).Primary().OnClick(func() { d.clear(kind) }),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).Padding(th.Spacing.S).
		Background(ui.TokenChromeMuted).
		Style(ui.Spec{}.Radius(th.Radius.Medium).Border(ui.TokenWarning, th.Stroke.Thin)).Shrink(0)
}

func (d *Dialog) editor(th *theme.Theme) ui.View {
	e := d.edit
	c := &e.cookie
	// Field ids include the cookie so switching cookies resets carets.
	key := c.ID
	if e.isNew {
		key = "new"
	}
	fid := func(name string) string { return "ck-" + name + "-" + key }
	changed := func() { e.err = "" }

	title := "Edit cookie"
	if e.isNew {
		title = "New cookie"
	}

	sameSite := 0
	for i, o := range sameSiteOptions {
		if o.Value == c.SameSite {
			sameSite = i
		}
	}

	rows := []ui.View{
		ui.Row(ui.Strong(title).Grow(1),
			ui.IconButton(fid("close"), icons.X).Tooltip("Close").OnClick(func() {
				d.edit = nil
				d.refresh()
			}),
		).Align(ui.AlignCenter),
		field(th, "Name", ui.TextField(fid("name"), c.Name).Placeholder("session_id").
			OnChange(func(s string) { c.Name = s; changed() })),
		field(th, "Value", ui.TextField(fid("value"), c.Value).
			OnChange(func(s string) { c.Value = s; changed() })),
		field(th, "Domain", ui.TextField(fid("domain"), c.Domain).Placeholder("api.example.com").
			OnChange(func(s string) { c.Domain = s; changed() })),
		field(th, "Path", ui.TextField(fid("path"), c.Path).Placeholder("/").
			OnChange(func(s string) { c.Path = s; changed() })),
		field(th, "Expires", ui.TextField(fid("expires"), e.expires).Placeholder("Empty for a session cookie").
			OnChange(func(s string) { e.expires = s; changed() })),
		field(th, "SameSite", ui.Select(fid("samesite"), sameSiteOptions).Selected(sameSite).
			OnChange(func(v string) { c.SameSite = v; changed() })),
		ui.Row(
			ui.Column(
				ui.Checkbox(fid("enabled"), "Enabled").Check(c.Enabled).OnToggle(func(v bool) { c.Enabled = v }),
				ui.Checkbox(fid("secure"), "Secure").Check(c.Secure).OnToggle(func(v bool) { c.Secure = v }),
			).Gap(th.Spacing.S).Grow(1),
			ui.Column(
				ui.Checkbox(fid("hostonly"), "Host only").Check(c.HostOnly).OnToggle(func(v bool) { c.HostOnly = v }),
				ui.Checkbox(fid("httponly"), "HttpOnly").Check(c.HttpOnly).OnToggle(func(v bool) { c.HttpOnly = v }),
			).Gap(th.Spacing.S).Grow(1),
		).MarginTop(th.Spacing.XS),
	}
	if !e.isNew {
		rows = append(rows, ui.Caption(fmt.Sprintf("Source: %s · created %s", c.Source, c.Created.Local().Format(expiryDisplay))))
	}
	if e.err != "" {
		rows = append(rows, ui.Paragraph(e.err).Style(ui.Spec{}.TextColor(ui.TokenError)))
	}

	actions := []ui.View{ui.Spacer()}
	if !e.isNew {
		id := c.ID
		actions = []ui.View{
			ui.Button(fid("delete"), ui.Text("Delete")).IconStart(icons.Trash2).Subtle().OnClick(func() { d.deleteCookie(id) }),
			ui.Spacer(),
		}
	}
	actions = append(actions, ui.Button(fid("save"), ui.Text("Save")).Primary().OnClick(d.commitEdit))
	rows = append(rows, ui.Spacer(), ui.Row(actions...).Gap(th.Spacing.S).Align(ui.AlignCenter))

	// The panel scrolls on its own; letting it stretch the dialog body would
	// shrink the toolbar and the table rows. The fixed width lives on the
	// wrapper: a scroll view always grows to the space a row gives it.
	return ui.Column(
		ui.Scroll("ck-editor-scroll",
			ui.Column(rows...).Gap(th.Spacing.S).Padding(th.Spacing.M),
		),
	).Width(320).Shrink(0).
		Background(ui.TokenChromeMuted).
		Style(ui.Spec{}.Radius(th.Radius.Medium).Border(ui.TokenBorder, th.Stroke.Thin))
}

func field(th *theme.Theme, label string, control *ui.Node) ui.View {
	return ui.Column(ui.Caption(label), control).Gap(th.Spacing.XXS)
}
