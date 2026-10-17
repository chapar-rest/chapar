package pages

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/mitm"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui/container"
	"github.com/chapar-rest/chapar/version"
)

// proxyRebuildEvery throttles rebuilding the table while traffic streams in.
const proxyRebuildEvery = 150 * time.Millisecond

// ProxyDeps wires the proxy page to the app.
type ProxyDeps struct {
	Service *mitm.Service
	// Container gives the response editors their Save/Copy menu.
	Container func() container.Deps
	Error     func(error)
	Toast     func(string)
	// OpenSettings opens the settings dialog.
	OpenSettings func()
	// Wake asks for a frame from any goroutine.
	Wake func()
}

// scopeKind is what the sidebar selection narrows the flows to.
type scopeKind int

const (
	scopeAll scopeKind = iota
	scopeHost
	scopeApp
	scopePinned
)

type proxyScope struct {
	kind  scopeKind
	value string
}

// proxyNode is the Data of a sidebar tree node.
type proxyNode struct {
	group string // "pinned", "apps", "domains" for the group rows
	scope proxyScope
}

// filterChip is one of the buttons above the table.
type filterChip struct {
	id    string
	label string
	match func(f *mitm.Flow) bool
}

var protocolChips = []filterChip{
	{"http", "HTTP", func(f *mitm.Flow) bool { return !f.HTTPS() && f.Kind != mitm.KindWS }},
	{"https", "HTTPS", func(f *mitm.Flow) bool { return f.HTTPS() }},
	{"ws", "WS", func(f *mitm.Flow) bool { return f.Kind == mitm.KindWS }},
}

var contentChips = []filterChip{
	{"json", "JSON", kindIs(mitm.KindJSON)},
	{"xml", "XML", kindIs(mitm.KindXML)},
	{"html", "HTML", kindIs(mitm.KindHTML)},
	{"css", "CSS", kindIs(mitm.KindCSS)},
	{"js", "JS", kindIs(mitm.KindJS)},
	{"img", "Images", kindIs(mitm.KindImage)},
	{"media", "Media", kindIs(mitm.KindMedia)},
	{"font", "Fonts", kindIs(mitm.KindFont)},
	{"form", "Forms", kindIs(mitm.KindForm)},
	{"other", "Other", kindIs(mitm.KindText, mitm.KindOther, mitm.KindTunnel, "")},
}

var errorChip = filterChip{"err", "Errors", func(f *mitm.Flow) bool {
	return f.State == mitm.StateFailed || f.Status >= 400
}}

func kindIs(kinds ...mitm.Kind) func(*mitm.Flow) bool {
	return func(f *mitm.Flow) bool { return slices.Contains(kinds, f.Kind) }
}

// Proxy is the traffic capture page: a sidebar of apps and domains, the
// captured flows in a table, and the selected flow's request and
// response side by side.
type Proxy struct {
	deps     ProxyDeps
	SideOpen bool

	tree    *ui.Tree
	treeSig string
	table   *ui.Table

	scope     proxyScope
	chips     map[string]bool
	query     string
	treeQuery string

	seenVersion uint64
	lastRebuild time.Time
	dirty       bool
	shown       []*mitm.Flow

	selID   uint64
	reqTab  int
	respTab int
	reqEd   *ui.Editor
	respEd  *ui.Editor
	edFlow  *mitm.Flow
	edTabs  [2]int

	setup *proxySetup
}

// NewProxyPage builds the proxy page.
func NewProxyPage(deps ProxyDeps) *Proxy {
	p := &Proxy{deps: deps, SideOpen: true, chips: map[string]bool{}, dirty: true}
	p.setup = newProxySetup(p)

	p.tree = ui.NewTree(&ui.TreeNode{Label: "root"})
	p.tree.IconFor = p.iconFor
	p.tree.OnActivate = p.activate
	p.tree.ContextMenu = p.treeMenu

	p.table = ui.NewTable([]ui.TableColumn{
		{ID: "n", Label: "#", Kind: ui.TableColText, Width: 56, Locked: true},
		{ID: "method", Label: "Method", Kind: ui.TableColText, Width: 80, Sortable: true},
		{ID: "status", Label: "Status", Kind: ui.TableColText, Width: 64, Sortable: true},
		{ID: "host", Label: "Host", Kind: ui.TableColText, Width: 200, Sortable: true},
		{ID: "path", Label: "Path", Kind: ui.TableColText, Sortable: true},
		{ID: "type", Label: "Type", Kind: ui.TableColText, Width: 70, Sortable: true},
		{ID: "size", Label: "Size", Kind: ui.TableColText, Width: 80, Locked: true},
		{ID: "time", Label: "Time", Kind: ui.TableColText, Width: 76, Locked: true},
		{ID: "app", Label: "App", Kind: ui.TableColText, Width: 120, Sortable: true},
	}, nil)
	p.table.Editable = false
	p.table.Selectable = true
	p.table.MinHeight = 120
	p.table.CellHighlight = p.cellColor
	p.table.OnRowClick = func(rowID string) {
		id, err := strconv.ParseUint(rowID, 10, 64)
		if err == nil {
			p.selID = id
			p.dirty = true
		}
	}
	return p
}

// Close releases the editors.
func (p *Proxy) Close() {
	if p.reqEd != nil {
		p.reqEd.Close()
	}
	if p.respEd != nil {
		p.respEd.Close()
	}
}

func (p *Proxy) svc() *mitm.Service { return p.deps.Service }

func (p *Proxy) wake() {
	if p.deps.Wake != nil {
		p.deps.Wake()
	}
}

// Start runs the proxy with the saved settings.
func (p *Proxy) Start() {
	cfg := prefs.GetGlobalConfig().Spec.Proxy
	host := "127.0.0.1"
	if cfg.AllowLAN {
		host = "0.0.0.0"
	}
	err := p.svc().Start(mitm.Config{
		ListenAddr:   fmt.Sprintf("%s:%d", host, cfg.ListenPort()),
		MaxFlows:     cfg.FlowLimit(),
		MaxBodyBytes: cfg.BodyLimitBytes(),
		IgnoreHosts:  cfg.IgnoreHosts,
	})
	if err != nil {
		p.deps.Error(err)
		return
	}
	p.setup.recheck()
}

// Stop shuts the proxy down.
func (p *Proxy) Stop() {
	if err := p.svc().Stop(); err != nil {
		p.deps.Error(err)
	}
}

// Clear drops every captured flow.
func (p *Proxy) Clear() {
	p.svc().Store().Clear()
	p.selID = 0
	p.dirty = true
}

// ExportHAR saves the requests the table shows, in the order they were
// sent, as a HAR file.
func (p *Proxy) ExportHAR() {
	data, err := mitm.HAR(p.shown, version.GetAppVersion())
	if err != nil {
		p.deps.Error(err)
		return
	}
	files := p.deps.Container().Files
	if files == nil {
		return
	}
	fd := files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title: "Export HAR",
		Mode:  ui.FileDialogSaveFile,
		Filters: []ui.FileFilter{
			{Label: "HAR files", Exts: []string{".har"}},
			{Label: "All files", Exts: nil},
		},
		ShowSaveFilter: true,
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			path := paths[0]
			if filepath.Ext(path) == "" {
				path += ".har"
			}
			if err := os.WriteFile(path, data, 0o644); err != nil {
				p.deps.Error(err)
				return
			}
			p.deps.Toast("Exported " + filepath.Base(path))
		},
	})
}

// ShowSetup opens the readiness checklist.
func (p *Proxy) ShowSetup(c *ui.Ctx) { p.setup.show(c) }

// listenAddr is the address clients should use, for display.
func (p *Proxy) listenAddr() string {
	cfg := prefs.GetGlobalConfig().Spec.Proxy
	return fmt.Sprintf("127.0.0.1:%d", cfg.ListenPort())
}

// --- data ---------------------------------------------------------------

// refresh rebuilds the table and sidebar when the store changed, at most
// every proxyRebuildEvery so a burst of traffic doesn't rebuild per flow.
func (p *Proxy) refresh(c *ui.Ctx) {
	v := p.svc().Store().Version()
	if v != p.seenVersion {
		p.dirty = true
	}
	if !p.dirty {
		return
	}
	if wait := proxyRebuildEvery - time.Since(p.lastRebuild); wait > 0 {
		c.Animate(wait)
		return
	}
	p.seenVersion = v
	p.dirty = false
	p.lastRebuild = time.Now()

	flows := p.svc().Store().Snapshot()
	p.rebuildTree(flows)

	shown := make([]*mitm.Flow, 0, len(flows))
	for i := len(flows) - 1; i >= 0; i-- {
		if p.matches(flows[i]) {
			shown = append(shown, flows[i])
		}
	}
	p.shown = shown

	rows := make([]ui.TableRow, len(shown))
	for i, f := range shown {
		rows[i] = ui.TableRow{
			ID:       strconv.FormatUint(f.ID, 10),
			Selected: f.ID == p.selID,
			Icon:     flowIcon(f),
			Cells: map[string]string{
				"n":      strconv.FormatUint(f.ID, 10),
				"method": f.Method,
				"status": statusCell(f),
				"host":   f.Host,
				"path":   pathCell(f),
				"type":   string(f.Kind),
				"size":   sizeCell(f),
				"time":   timeCell(f),
				"app":    f.App,
			},
		}
	}
	p.table.SetRows(rows)

	// A pending flow's elapsed time keeps counting.
	for _, f := range shown {
		if f.State == mitm.StatePending {
			c.Animate(time.Second)
			break
		}
	}
}

func (p *Proxy) matches(f *mitm.Flow) bool {
	switch p.scope.kind {
	case scopeHost:
		if f.Host != p.scope.value {
			return false
		}
	case scopeApp:
		if f.App != p.scope.value {
			return false
		}
	case scopePinned:
		st := proxyState()
		if !slices.Contains(st.PinnedHosts, f.Host) && !slices.Contains(st.PinnedApps, f.App) {
			return false
		}
	}
	if !chipsMatch(p.chips, protocolChips, f) || !chipsMatch(p.chips, contentChips, f) {
		return false
	}
	if p.chips[errorChip.id] && !errorChip.match(f) {
		return false
	}
	if q := strings.ToLower(strings.TrimSpace(p.query)); q != "" {
		if !strings.Contains(strings.ToLower(f.URL), q) &&
			!strings.Contains(strings.ToLower(f.Method), q) &&
			!strings.Contains(strings.ToLower(f.App), q) &&
			!strings.Contains(strconv.Itoa(f.Status), q) {
			return false
		}
	}
	return true
}

// chipsMatch reports whether f matches one of the chips of a group that
// are on, or true when none of them are.
func chipsMatch(on map[string]bool, group []filterChip, f *mitm.Flow) bool {
	any := false
	for _, ch := range group {
		if !on[ch.id] {
			continue
		}
		any = true
		if ch.match(f) {
			return true
		}
	}
	return !any
}

func (p *Proxy) selected() *mitm.Flow {
	if p.selID == 0 {
		return nil
	}
	f, _ := p.svc().Store().Get(p.selID)
	return f
}

// --- sidebar ------------------------------------------------------------

func proxyState() domain.ProxyState {
	st := prefs.GetAppState()
	if st.Spec.Proxy == nil {
		return domain.ProxyState{}
	}
	return *st.Spec.Proxy
}

func updateProxyState(fn func(*domain.ProxyState)) error {
	st := prefs.GetAppState()
	ps := domain.ProxyState{}
	if st.Spec.Proxy != nil {
		ps = *st.Spec.Proxy
	}
	fn(&ps)
	st.Spec.Proxy = &ps
	return prefs.UpdateAppState(st)
}

func (p *Proxy) rebuildTree(flows []*mitm.Flow) {
	hosts := map[string]bool{}
	apps := map[string]bool{}
	for _, f := range flows {
		if f.Host != "" {
			hosts[f.Host] = true
		}
		if f.App != "" {
			apps[f.App] = true
		}
	}
	st := proxyState()
	hostList := sortedKeys(hosts)
	appList := sortedKeys(apps)

	sig := strings.Join(st.PinnedApps, ",") + "|" + strings.Join(st.PinnedHosts, ",") + "|" +
		strings.Join(appList, ",") + "|" + strings.Join(hostList, ",")
	if sig == p.treeSig {
		return
	}
	first := p.treeSig == ""
	p.treeSig = sig

	open := map[string]bool{"pinned": true, "apps": true, "domains": true}
	if !first {
		open = map[string]bool{}
		if root := p.tree.Root(); root != nil {
			for _, n := range root.Children {
				if d, ok := n.Data.(proxyNode); ok && d.group != "" && n.Expanded() {
					open[d.group] = true
				}
			}
		}
	}

	leaf := func(kind scopeKind, v string) *ui.TreeNode {
		return &ui.TreeNode{Label: v, Leaf: true, Data: proxyNode{scope: proxyScope{kind, v}}}
	}
	group := func(id, label string, kids []*ui.TreeNode) *ui.TreeNode {
		return &ui.TreeNode{Label: label, Data: proxyNode{group: id}, Children: kids}
	}

	var pinned []*ui.TreeNode
	for _, a := range st.PinnedApps {
		pinned = append(pinned, leaf(scopeApp, a))
	}
	for _, h := range st.PinnedHosts {
		pinned = append(pinned, leaf(scopeHost, h))
	}
	var appNodes, hostNodes []*ui.TreeNode
	for _, a := range appList {
		appNodes = append(appNodes, leaf(scopeApp, a))
	}
	for _, h := range hostList {
		hostNodes = append(hostNodes, leaf(scopeHost, h))
	}

	root := &ui.TreeNode{Label: "root"}
	root.Children = []*ui.TreeNode{
		{Label: "All traffic", Leaf: true, Data: proxyNode{scope: proxyScope{kind: scopeAll}}},
	}
	if len(pinned) > 0 {
		root.Children = append(root.Children, group("pinned", "Pinned", pinned))
	}
	root.Children = append(root.Children,
		group("apps", "Apps", appNodes),
		group("domains", "Domains", hostNodes),
	)
	p.tree.SetRoot(root)
	for _, n := range root.Children {
		if d, ok := n.Data.(proxyNode); ok && open[d.group] {
			p.tree.SetExpanded(n, true)
		}
	}
	p.tree.SetFilter(p.treeQuery)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (p *Proxy) iconFor(n *ui.TreeNode, expanded bool) (icons.Icon, render.Color) {
	th := theme.Current()
	d, _ := n.Data.(proxyNode)
	switch {
	case d.group == "pinned":
		return icons.Pin, th.ForegroundMuted
	case d.group == "apps":
		return icons.AppWindow, th.ForegroundMuted
	case d.group == "domains":
		return icons.Globe, th.ForegroundMuted
	case d.scope.kind == scopeAll:
		return icons.Activity, th.Accent
	case d.scope.kind == scopeApp:
		if strings.HasPrefix(d.scope.value, "Device ") {
			return icons.Smartphone, th.ForegroundMuted
		}
		return icons.AppWindow, th.ForegroundMuted
	}
	return icons.Globe, th.ForegroundMuted
}

func (p *Proxy) activate(n *ui.TreeNode) {
	d, ok := n.Data.(proxyNode)
	if !ok || d.group != "" {
		return
	}
	p.scope = d.scope
	p.dirty = true
}

func (p *Proxy) treeMenu(n *ui.TreeNode) []ui.MenuItem {
	if n == nil {
		return nil
	}
	d, ok := n.Data.(proxyNode)
	if !ok {
		return nil
	}
	if d.group == "pinned" {
		return []ui.MenuItem{{Label: "Show all pinned", OnSelect: func() {
			p.scope = proxyScope{kind: scopePinned}
			p.dirty = true
		}}}
	}
	v := d.scope.value
	switch d.scope.kind {
	case scopeHost:
		st := proxyState()
		pinned := slices.Contains(st.PinnedHosts, v)
		items := []ui.MenuItem{
			{Label: pinLabel(pinned), OnSelect: func() { p.togglePin(scopeHost, v) }},
		}
		cfg := prefs.GetGlobalConfig().Spec.Proxy
		if mitm.MatchHost(cfg.IgnoreHosts, v) {
			items = append(items, ui.MenuItem{Label: "Decrypt this domain again", OnSelect: func() { p.setIgnored(v, false) }})
		} else {
			items = append(items, ui.MenuItem{Label: "Don't decrypt this domain", OnSelect: func() { p.setIgnored(v, true) }})
		}
		return append(items,
			ui.MenuSeparator,
			ui.MenuItem{Label: "Remove its requests", OnSelect: func() { p.removeWhere(func(f *mitm.Flow) bool { return f.Host == v }) }},
		)
	case scopeApp:
		pinned := slices.Contains(proxyState().PinnedApps, v)
		return []ui.MenuItem{
			{Label: pinLabel(pinned), OnSelect: func() { p.togglePin(scopeApp, v) }},
			ui.MenuSeparator,
			{Label: "Remove its requests", OnSelect: func() { p.removeWhere(func(f *mitm.Flow) bool { return f.App == v }) }},
		}
	}
	return nil
}

func pinLabel(pinned bool) string {
	if pinned {
		return "Unpin"
	}
	return "Pin"
}

func (p *Proxy) togglePin(kind scopeKind, v string) {
	err := updateProxyState(func(ps *domain.ProxyState) {
		list := &ps.PinnedHosts
		if kind == scopeApp {
			list = &ps.PinnedApps
		}
		if i := slices.Index(*list, v); i >= 0 {
			*list = slices.Delete(*list, i, i+1)
		} else {
			*list = append(*list, v)
		}
	})
	if err != nil {
		p.deps.Error(err)
	}
	p.dirty = true
}

// setIgnored adds host to, or removes it from, the hosts passed through
// without decrypting, and applies it to the running proxy.
func (p *Proxy) setIgnored(host string, ignore bool) {
	cfg := prefs.GetGlobalConfig()
	list := slices.DeleteFunc(slices.Clone(cfg.Spec.Proxy.IgnoreHosts), func(h string) bool {
		return strings.EqualFold(h, host)
	})
	if ignore {
		list = append(list, host)
	}
	cfg.Spec.Proxy.IgnoreHosts = list
	if err := prefs.UpdateGlobalConfig(cfg); err != nil {
		p.deps.Error(err)
		return
	}
	p.svc().SetIgnoreHosts(list)
	if ignore {
		p.deps.Toast(host + " now passes through without decrypting")
	} else {
		p.deps.Toast(host + " will be decrypted again")
	}
}

func (p *Proxy) removeWhere(match func(*mitm.Flow) bool) {
	ids := map[uint64]bool{}
	for _, f := range p.svc().Store().Snapshot() {
		if match(f) {
			ids[f.ID] = true
		}
	}
	p.svc().Store().Remove(ids)
	if ids[p.selID] {
		p.selID = 0
	}
	p.dirty = true
}

// --- table cells ----------------------------------------------------------

func flowIcon(f *mitm.Flow) icons.Icon {
	switch {
	case f.State == mitm.StateFailed:
		return icons.CircleAlert
	case f.State == mitm.StatePending:
		return icons.Loader
	case f.Tunnel:
		return icons.ArrowRightLeft
	case f.Kind == mitm.KindWS:
		return icons.Waypoints
	case f.HTTPS():
		return icons.Lock
	}
	return icons.LockOpen
}

func statusCell(f *mitm.Flow) string {
	switch {
	case f.State == mitm.StateFailed:
		return "Failed"
	case f.State == mitm.StatePending:
		return "…"
	case f.Tunnel:
		return "Tunnel"
	}
	return strconv.Itoa(f.Status)
}

func pathCell(f *mitm.Flow) string {
	if f.Tunnel {
		return "(not decrypted)"
	}
	return f.Path
}

func sizeCell(f *mitm.Flow) string {
	switch {
	case f.State == mitm.StatePending || f.Tunnel:
		return ""
	case f.RespStreamed && f.RespSize == 0:
		return "streamed"
	}
	return container.FormatBytes(int(f.RespSize))
}

func timeCell(f *mitm.Flow) string {
	d := f.Duration()
	if d < time.Second {
		return strconv.FormatInt(d.Milliseconds(), 10) + " ms"
	}
	return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + " s"
}

func (p *Proxy) cellColor(_, colID, value string) []ui.TextSpan {
	if colID != "status" || value == "" {
		return nil
	}
	th := theme.Current()
	var c render.Color
	switch {
	case value == "Failed" || strings.HasPrefix(value, "5"):
		c = th.Error
	case strings.HasPrefix(value, "4"):
		c = th.Warning
	case strings.HasPrefix(value, "2"):
		c = th.Success
	case strings.HasPrefix(value, "3"):
		c = th.Accent
	default:
		return nil
	}
	return []ui.TextSpan{{Start: 0, End: len(value), Color: c}}
}

// --- layout ---------------------------------------------------------------

func (p *Proxy) Layout(c *ui.Ctx) ui.View {
	p.refresh(c)
	main := p.mainPane(c)
	var body ui.View = main
	if p.SideOpen {
		body = ui.Splitter("proxy-side-split", ui.Horizontal, p.side(c), main).
			Sizes(240, 0).
			HandleOnHover().
			Grow(1)
	}
	th := c.Theme()
	return ui.Column(
		p.header(c),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(body).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (p *Proxy) header(c *ui.Ctx) ui.View {
	th := c.Theme()
	svc := p.svc()
	running := svc.Running()

	var status ui.View
	if running {
		status = ui.Row(
			ui.Badge("Listening").Tone(ui.BadgeSuccess),
			ui.Caption(displayAddr(svc.Addr())),
		).Gap(th.Spacing.S).Align(ui.AlignCenter)
	} else {
		status = ui.Badge("Stopped")
	}

	startBtn := ui.Button("proxy-start", ui.Text("Start")).Primary().IconStart(icons.Play).OnClick(p.Start)
	if running {
		startBtn = ui.Button("proxy-stop", ui.Text("Stop")).IconStart(icons.Square).OnClick(p.Stop)
	}

	pauseLabel, pauseIcon := "Pause", icons.Pause
	if svc.Paused() {
		pauseLabel, pauseIcon = "Resume", icons.Play
	}

	return ui.Row(
		ui.Title("Proxy"),
		status,
		ui.Spacer(),
		p.setup.button(c),
		ui.Button("proxy-pause", ui.Text(pauseLabel)).IconStart(pauseIcon).Ghost().HoverFill().
			Tooltip("Stop recording new requests; traffic keeps flowing").
			OnClick(func() { svc.SetPaused(!svc.Paused()) }),
		ui.Button("proxy-export", ui.Text("Export HAR")).IconStart(icons.Download).Ghost().HoverFill().
			Disabled(len(p.shown) == 0).
			Tooltip("Save the requests shown as a HAR file").
			OnClick(p.ExportHAR),
		ui.Button("proxy-clear", ui.Text("Clear")).IconStart(icons.Eraser).Ghost().HoverFill().OnClick(p.Clear),
		startBtn,
	).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.M, th.Spacing.S)
}

// displayAddr shows a wildcard listen address as the loopback one.
func displayAddr(addr string) string {
	return strings.Replace(addr, "0.0.0.0", "127.0.0.1", 1)
}

func (p *Proxy) side(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.TextField("proxy-tree-search", p.treeQuery).Placeholder("Filter apps and domains").
			IconStart(icons.Search).
			OnChange(func(s string) { p.treeQuery = s; p.tree.SetFilter(s) }).
			Margin(th.Spacing.S),
		ui.ViewOf(p.tree).Grow(1),
	).Gap(th.Spacing.XS).Grow(1).Background(ui.TokenChrome)
}

func (p *Proxy) mainPane(c *ui.Ctx) ui.View {
	th := c.Theme()
	splitDir := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)
	detail := ui.Splitter("proxy-detail-split", splitDir, p.reqPane(c), p.respPane(c)).
		Percents(50, 50).
		HandleOnHover().
		Grow(1)
	return ui.Column(
		p.filterBar(c),
		ui.Splitter("proxy-main-split", ui.Vertical, p.listPane(c), detail).
			Percents(55, 45).
			HandleOnHover().
			Grow(1),
	).Gap(th.Spacing.XS).Grow(1)
}

func (p *Proxy) filterBar(c *ui.Ctx) ui.View {
	th := c.Theme()
	chip := func(ch filterChip) ui.View {
		id := ch.id
		return chipButton("proxy-chip-"+id, ch.label, p.chips[id]).OnClick(func() {
			p.chips[id] = !p.chips[id]
			p.dirty = true
		})
	}
	all := chipButton("proxy-chip-all", "All", len(p.activeChips()) == 0)
	views := []ui.View{all.OnClick(func() {
		p.chips = map[string]bool{}
		p.dirty = true
	})}
	for _, ch := range protocolChips {
		views = append(views, chip(ch))
	}
	views = append(views, ui.VLine(th.Stroke.Thin, th.Border).Height(18))
	for _, ch := range contentChips {
		views = append(views, chip(ch))
	}
	views = append(views, ui.VLine(th.Stroke.Thin, th.Border).Height(18), chip(errorChip),
		ui.TextField("proxy-search", p.query).Placeholder("Filter URL, method, status, app").
			IconStart(icons.Search).Grow(1).
			OnChange(func(s string) { p.query = s; p.dirty = true }),
	)
	return ui.Row(views...).Gap(th.Spacing.XXS).Align(ui.AlignCenter).Wrap().
		PaddingXY(th.Spacing.S, th.Spacing.XS)
}

// chipButton is a filter button. On and off differ only in color: button
// variants have different padding, so switching variant would resize it.
func chipButton(id, label string, on bool) *ui.Node {
	b := ui.Button(id, ui.Text(label)).Ghost().HoverFill()
	if on {
		b = b.Style(ui.Background(ui.TokenListActive).TextColor(ui.TokenForeground))
	}
	return b
}

func (p *Proxy) activeChips() []string {
	var out []string
	for k, v := range p.chips {
		if v {
			out = append(out, k)
		}
	}
	return out
}

func (p *Proxy) listPane(c *ui.Ctx) ui.View {
	th := c.Theme()
	if len(p.shown) > 0 {
		return ui.ViewOf(p.table).Grow(1)
	}
	svc := p.svc()
	var empty *ui.Node
	switch {
	case !svc.Running():
		empty = ui.EmptyState("The proxy is off",
			"Start it, then point an app, browser or phone at "+p.listenAddr()+" to see its traffic here.").
			EmptyIcon(icons.Radar)
	case svc.Store().Len() > 0:
		empty = ui.EmptyState("No matches", "No request matches the selection and filters.").EmptyIcon(icons.Search)
	default:
		empty = ui.EmptyState("Waiting for traffic",
			"Point an app at "+displayAddr(svc.Addr())+". To trust Chapar's certificate on a phone or another "+
				"computer, open http://"+mitm.CertSiteHost+" through the proxy.").
			EmptyIcon(icons.Radar)
	}
	actions := []ui.View{}
	if !svc.Running() {
		actions = append(actions, ui.Button("proxy-empty-start", ui.Text("Start proxy")).Primary().IconStart(icons.Play).OnClick(p.Start))
	}
	actions = append(actions, ui.Button("proxy-empty-setup", ui.Text("Setup guide")).IconStart(icons.ShieldCheck).
		OnClick(func() { p.ShowSetup(c) }))
	return ui.Column(
		empty,
		ui.Row(actions...).Gap(th.Spacing.S).Justify(ui.JustifyCenter).PaddingBottom(th.Spacing.M),
	).Gap(th.Spacing.M).Justify(ui.JustifyCenter).Grow(1)
}

// --- detail panes -----------------------------------------------------------

var reqTabs = []ui.TabModel{{Title: "Headers"}, {Title: "Query"}, {Title: "Body"}, {Title: "Raw"}}
var respTabs = []ui.TabModel{{Title: "Headers"}, {Title: "Body"}, {Title: "Raw"}}

// syncEditors rebuilds the editors when the selected flow, its content or
// a tab changed.
func (p *Proxy) syncEditors(f *mitm.Flow) {
	tabs := [2]int{p.reqTab, p.respTab}
	if f == p.edFlow && tabs == p.edTabs && p.reqEd != nil {
		return
	}
	p.edFlow, p.edTabs = f, tabs
	if f == nil {
		p.reqEd = container.ReplaceEditor(p.reqEd, nil, highlight.Noop{})
		p.respEd = container.ReplaceEditor(p.respEd, nil, highlight.Noop{})
		return
	}
	reqData, reqHL := requestTab(f, p.reqTab)
	p.reqEd = container.ReplaceEditor(p.reqEd, reqData, reqHL)
	respData, respHL := responseTab(f, p.respTab)
	p.respEd = container.ReplaceEditor(p.respEd, respData, respHL)
}

func (p *Proxy) reqPane(c *ui.Ctx) ui.View {
	th := c.Theme()
	f := p.selected()
	p.syncEditors(f)
	if f == nil {
		return paneCard(th, container.MutedParagraph("Select a request to see it here."))
	}
	title := f.Method + " " + f.URL
	return paneCard(th,
		ui.Paragraph(title).Style(ui.Spec{}.TextColor(ui.TokenForeground)),
		ui.Tabs("proxy-req-tabs", reqTabs).
			Selected(p.reqTab).Closable(false).
			OnSelectItem(func(i int, _ string) { p.reqTab = i }).
			TabBackground(th.Background),
		container.ResponseEditorMenu(p.reqEd, c, p.deps.Container(), "request.txt"),
	)
}

func (p *Proxy) respPane(c *ui.Ctx) ui.View {
	th := c.Theme()
	f := p.selected()
	if f == nil {
		return paneCard(th, container.MutedParagraph("The response shows here."))
	}
	var line ui.View
	switch {
	case f.State == mitm.StatePending:
		line = ui.Caption("Waiting for the response…")
	case f.State == mitm.StateFailed:
		line = ui.Paragraph(f.Error).Style(ui.Spec{}.TextColor(ui.TokenError))
	case f.Tunnel:
		line = container.MutedParagraph("Passed through without decrypting, so there is nothing to show. " +
			"Remove the domain from the ignore list to see its requests.")
	default:
		text := fmt.Sprintf("%d %s · %s · %s", f.Status, http.StatusText(f.Status),
			container.FormatBytes(int(f.RespSize)), timeCell(f))
		line = ui.Text(text).Style(container.StatusLineStyle(false, f.Status, true))
	}
	if f.State != mitm.StateDone || f.Tunnel {
		return paneCard(th, line)
	}
	name := "response.txt"
	if ext := kindExt(f.Kind); ext != "" && p.respTab == 1 {
		name = "response." + ext
	}
	return paneCard(th,
		line,
		ui.Tabs("proxy-resp-tabs", respTabs).
			Selected(p.respTab).Closable(false).
			OnSelectItem(func(i int, _ string) { p.respTab = i }).
			TabBackground(th.Background),
		container.ResponseEditorMenu(p.respEd, c, p.deps.Container(), name),
	)
}

func paneCard(th *theme.Theme, children ...ui.View) ui.View {
	return ui.Column(
		ui.Column(children...).
			Radius(th.Radius.Medium).
			Border(ui.TokenBorder, th.Stroke.Thick).
			BorderStyle(ui.BorderDotted).
			Margin(th.Spacing.XS).
			Padding(th.Spacing.S).
			Gap(th.Spacing.XS).
			Grow(1),
	)
}

func requestTab(f *mitm.Flow, tab int) ([]byte, highlight.Highlighter) {
	switch tab {
	case 0:
		return headerText(f.ReqHeader), highlight.Noop{}
	case 1:
		u, err := url.Parse(f.URL)
		if err != nil || u.RawQuery == "" {
			return []byte("No query parameters."), highlight.Noop{}
		}
		var b strings.Builder
		q := u.Query()
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range q[k] {
				b.WriteString(k + " = " + v + "\n")
			}
		}
		return []byte(b.String()), highlight.Noop{}
	case 2:
		if f.ReqStreamed {
			return []byte("The body was larger than the capture limit and was not kept."), highlight.Noop{}
		}
		kind := mitm.Classify(f.ReqHeader.Get("Content-Type"), "")
		return bodyText(f.ReqBody, kind, f.ReqHeader.Get("Content-Type"))
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "%s %s %s\n", f.Method, rawTarget(f), f.Proto)
		b.Write(headerText(f.ReqHeader))
		if len(f.ReqBody) > 0 {
			b.WriteString("\n")
			b.Write(printable(f.ReqBody))
		}
		return []byte(b.String()), highlight.Noop{}
	}
}

func responseTab(f *mitm.Flow, tab int) ([]byte, highlight.Highlighter) {
	switch tab {
	case 0:
		return headerText(f.RespHeader), highlight.Noop{}
	case 1:
		if f.RespStreamed {
			return []byte("The body was larger than the capture limit and was not kept."), highlight.Noop{}
		}
		return bodyText(f.RespBody, f.Kind, f.RespHeader.Get("Content-Type"))
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "%s %d %s\n", f.Proto, f.Status, http.StatusText(f.Status))
		b.Write(headerText(f.RespHeader))
		if len(f.RespBody) > 0 {
			b.WriteString("\n")
			b.Write(printable(f.RespBody))
		}
		return []byte(b.String()), highlight.Noop{}
	}
}

func rawTarget(f *mitm.Flow) string {
	if f.Tunnel {
		return f.URL
	}
	return f.Path
}

func headerText(h http.Header) []byte {
	if len(h) == 0 {
		return []byte("No headers.")
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		for _, v := range h[k] {
			b.WriteString(k + ": " + v + "\n")
		}
	}
	return []byte(b.String())
}

// bodyText formats a body for reading: JSON pretty-printed, binary
// summarized instead of shown.
func bodyText(body []byte, kind mitm.Kind, ct string) ([]byte, highlight.Highlighter) {
	if len(body) == 0 {
		return []byte("No body."), highlight.Noop{}
	}
	if !isText(body, kind) {
		return []byte(fmt.Sprintf("Binary body, %s (%s). Right-click › Save… to keep it.",
			container.FormatBytes(len(body)), orDash(ct))), highlight.Noop{}
	}
	switch kind {
	case mitm.KindJSON:
		if pretty, err := container.FormatJSONBody(string(body), container.EditorIndent()); err == nil {
			return []byte(pretty), highlight.NewJSON()
		}
		return body, highlight.NewJSON()
	case mitm.KindXML, mitm.KindHTML:
		return body, highlight.NewXML()
	}
	return body, highlight.Noop{}
}

func isText(b []byte, kind mitm.Kind) bool {
	switch kind {
	case mitm.KindImage, mitm.KindMedia, mitm.KindFont:
		return false
	}
	return utf8.Valid(b[:min(len(b), 4096)]) || utf8.Valid(b)
}

// printable keeps raw views readable when a body is binary.
func printable(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	return []byte(fmt.Sprintf("<%s of binary data>", container.FormatBytes(len(b))))
}

func orDash(s string) string {
	if s == "" {
		return "no content type"
	}
	return s
}

func kindExt(k mitm.Kind) string {
	switch k {
	case mitm.KindJSON:
		return "json"
	case mitm.KindXML:
		return "xml"
	case mitm.KindHTML:
		return "html"
	case mitm.KindCSS:
		return "css"
	case mitm.KindJS:
		return "js"
	}
	return ""
}
