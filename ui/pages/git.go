package pages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/gitx"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui/container"
	"github.com/chapar-rest/chapar/ui/langsrv"
)

const (
	// gitTimeout bounds reading the repository, so a hung git (a network
	// mount, a stuck hook) can't leave the page loading forever.
	gitTimeout = 30 * time.Second
	// gitActionTimeout bounds a commit, which runs the user's hooks, and
	// the commands that talk to a remote.
	gitActionTimeout = 5 * time.Minute
	// gitLogLimit is how many commits the history shows.
	gitLogLimit = 200
	// gitDiffLimit caps the diff the editor shows.
	gitDiffLimit = 2 << 20
)

// GitDeps wires the git page to the app.
type GitDeps struct {
	// Dir is the open workspace's directory.
	Dir func() (string, error)
	// Open opens the item with id in a tab. It reports false when the
	// workspace has no such item.
	Open func(kind, id string) bool
	// Container gives the diff editors their Copy/Save menu.
	Container func() container.Deps
	Dialogs   func() *ui.DialogHost
	Error     func(error)
	Toast     func(string)
	// Wake asks for a frame from any goroutine.
	Wake func()
	// HasDirty reports whether any tab has unsaved changes.
	HasDirty func() bool
	// Reload reads the workspace from disk again after git changed its
	// files, and reopens the open tabs from the new files.
	Reload func()
	// DataDir is the folder that holds every workspace.
	DataDir func() string
	// Workspaces lists the workspaces DataDir holds.
	Workspaces func() []*domain.Workspace
	// Cloned opens a workspace that was just cloned.
	Cloned func(*domain.Workspace)
}

// gitEntry is a changed file and the item it holds.
type gitEntry struct {
	change gitx.Change
	item   gitItem
}

// gitSnapshot is what one refresh read.
type gitSnapshot struct {
	err     error
	dir     string
	version string
	repo    *gitx.Repo
	status  *gitx.Status
	entries []gitEntry
	commits []gitx.Commit
	// remote and remoteURL name the repository's remote, or are "".
	remote, remoteURL string
}

// gitDiffView is a list of changed files and the selected file's diff:
// the work tree's changes on one tab, a commit's on the other.
type gitDiffView struct {
	table   *ui.Table
	entries []gitEntry
	sel     string
	// key names what the diff shows; a result for another key is stale.
	key     string
	diff    []byte
	loading bool
	err     error
	ed      *ui.Editor
	edKey   string
	// edSel is the file the editor shows.
	edSel string
}

// Git is a read-only view of the workspace's repository: its branch, the
// changed items with their diffs, and the history.
type Git struct {
	deps GitDeps

	mu      sync.Mutex
	pending []func()

	refreshSeq uint64
	loading    bool
	snap       *gitSnapshot

	tab     int
	changes gitDiffView
	// excluded holds the changed files the user unticked; every other
	// change is in the next commit.
	excluded map[string]bool
	message  string
	// busy names the action running, such as "Pushing…", or is "".
	busy string
	form gitForm

	commits    *ui.Table
	selCommit  string
	commitKey  string
	commitBusy bool
	commitErr  error
	history    gitDiffView
}

// NewGitPage builds the git page. It reads nothing until Refresh.
func NewGitPage(deps GitDeps) *Git {
	g := &Git{deps: deps, excluded: map[string]bool{}}
	g.changes.table = g.newFileTable(true, func(id string) { g.selectChange(id) })
	g.changes.table.OnCellChange = func(path, colID, value string) {
		if colID == "inc" {
			g.setIncluded(path, value == "1")
		}
	}
	g.history.table = g.newFileTable(false, func(id string) { g.selectCommitFile(id) })

	g.commits = ui.NewTable([]ui.TableColumn{
		{ID: "hash", Label: "Commit", Kind: ui.TableColText, Width: 104, Locked: true},
		{ID: "subject", Label: "Message", Kind: ui.TableColText},
		{ID: "author", Label: "Author", Kind: ui.TableColText, Width: 150},
		{ID: "date", Label: "Date", Kind: ui.TableColText, Width: 140, Locked: true},
	}, nil)
	g.commits.Editable = false
	g.commits.Selectable = true
	g.commits.MinHeight = 120
	g.commits.CellHighlight = func(_, colID, value string) []ui.TextSpan {
		if colID != "hash" {
			return nil
		}
		return []ui.TextSpan{{Start: 0, End: len(value), Color: theme.Current().ForegroundMuted}}
	}
	g.commits.OnRowClick = g.selectCommit
	return g
}

// newFileTable lists changed files. With include, each row has a box
// that picks it for the next commit.
func (g *Git) newFileTable(include bool, onClick func(string)) *ui.Table {
	var cols []ui.TableColumn
	if include {
		cols = append(cols, ui.TableColumn{
			ID: "inc", Kind: ui.TableColToggle, Width: 40, Locked: true,
			IconOn: icons.SquareCheck, IconOff: icons.Square,
			TooltipOn: "In the next commit", TooltipOff: "Left out of the next commit",
		})
	}
	cols = append(cols,
		ui.TableColumn{ID: "change", Label: "Change", Kind: ui.TableColText, Width: 120, Sortable: true},
		ui.TableColumn{ID: "item", Label: "Item", Kind: ui.TableColText, Sortable: true},
		ui.TableColumn{ID: "type", Label: "Type", Kind: ui.TableColText, Width: 110, Sortable: true},
		ui.TableColumn{ID: "file", Label: "File", Kind: ui.TableColText, Sortable: true},
	)
	t := ui.NewTable(cols, nil)
	t.Editable = false
	t.Selectable = true
	t.MinHeight = 80
	t.CellHighlight = changeColor
	t.OnRowClick = onClick
	return t
}

// Close releases the editors.
func (g *Git) Close() {
	for _, v := range []*gitDiffView{&g.changes, &g.history} {
		if v.ed != nil {
			v.ed.Close()
			v.ed = nil
		}
	}
}

func (g *Git) wake() {
	if g.deps.Wake != nil {
		g.deps.Wake()
	}
}

// do runs work off the UI goroutine. work returns the function that
// applies its result, which runs at the start of the next frame.
func (g *Git) do(work func(ctx context.Context) func()) {
	g.doFor(gitTimeout, work)
}

// doFor is do with a time limit other than gitTimeout.
func (g *Git) doFor(limit time.Duration, work func(ctx context.Context) func()) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
		apply := work(ctx)
		g.mu.Lock()
		g.pending = append(g.pending, apply)
		g.mu.Unlock()
		g.wake()
	}()
}

// applyPending runs the results that arrived since the last frame. It
// reports whether any did.
func (g *Git) applyPending() bool {
	g.mu.Lock()
	fns := g.pending
	g.pending = nil
	g.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
	return len(fns) > 0
}

// Refresh reads the repository again: status, items and history.
func (g *Git) Refresh() {
	g.refreshSeq++
	seq := g.refreshSeq
	g.loading = true
	dir, dirErr := "", error(nil)
	if g.deps.Dir != nil {
		dir, dirErr = g.deps.Dir()
	}
	if g.snap != nil && g.snap.dir != dir {
		// Another workspace: don't show the old one's changes meanwhile.
		g.snap = nil
	}
	g.do(func(ctx context.Context) func() {
		snap := readGit(ctx, dir, dirErr)
		return func() {
			if seq != g.refreshSeq {
				return
			}
			g.loading = false
			g.setSnapshot(snap)
		}
	})
}

// readGit reads everything the page shows.
func readGit(ctx context.Context, dir string, dirErr error) *gitSnapshot {
	snap := &gitSnapshot{dir: dir}
	if dirErr != nil {
		snap.err = dirErr
		return snap
	}
	git, err := gitx.Find(ctx)
	if err != nil {
		snap.err = err
		return snap
	}
	snap.version = git.Version()
	repo, err := git.Open(ctx, dir)
	if err != nil {
		snap.err = err
		return snap
	}
	snap.repo = repo
	if snap.remote, snap.remoteURL, err = repo.Remote(ctx); err != nil {
		snap.err = err
		return snap
	}
	if snap.status, err = repo.Status(ctx); err != nil {
		snap.err = err
		return snap
	}
	for _, c := range snap.status.Changes {
		snap.entries = append(snap.entries, gitEntry{change: c, item: itemFor(c.Path, readItemFile(ctx, repo, dir, c))})
	}
	sortEntries(snap.entries)
	if snap.commits, err = repo.Log(ctx, gitLogLimit); err != nil {
		snap.err = err
	}
	return snap
}

// readItemFile reads a changed file to name its item: from the work tree,
// or from the last commit when it was deleted.
func readItemFile(ctx context.Context, repo *gitx.Repo, dir string, c gitx.Change) []byte {
	if c.Kind == gitx.Deleted {
		data, _ := repo.ShowFile(ctx, "HEAD", c.Path)
		return data
	}
	data, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(c.Path)))
	return data
}

// sortEntries orders the list by item type, then name.
func sortEntries(es []gitEntry) {
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i].item, es[j].item
		if a.Type != b.Type {
			return typeRank(a.Type) < typeRank(b.Type)
		}
		return a.label() < b.label()
	})
}

func typeRank(t string) int {
	for i, s := range []string{"Workspace", "Collection", "Request", "Environment", "Test case", "Proto file"} {
		if s == t {
			return i
		}
	}
	return 99
}

func (g *Git) setSnapshot(snap *gitSnapshot) {
	g.snap = snap
	// Forget unticked files that are no longer changed, so they are in
	// the commit if they change again.
	for p := range g.excluded {
		if findEntry(snap.entries, p) == nil {
			delete(g.excluded, p)
		}
	}
	g.changes.entries = snap.entries
	g.changes.table.SetRows(entryRows(snap.entries, g.changes.sel, g.excluded))
	if g.changes.sel != "" {
		// Reload the selected file's diff: it may have changed on disk.
		if findEntry(snap.entries, g.changes.sel) != nil {
			g.selectChange(g.changes.sel)
		} else {
			g.changes.clear()
		}
	}

	rows := make([]ui.TableRow, len(snap.commits))
	found := false
	for i, c := range snap.commits {
		rows[i] = ui.TableRow{
			ID:       c.Hash,
			Selected: c.Hash == g.selCommit,
			Icon:     icons.GitCommitHorizontal,
			Cells: map[string]string{
				"hash":    c.Short,
				"subject": c.Subject,
				"author":  c.Author,
				"date":    c.When.Format("2006-01-02 15:04"),
			},
		}
		found = found || c.Hash == g.selCommit
	}
	g.commits.SetRows(rows)
	if !found {
		g.selCommit = ""
		g.history.entries = nil
		g.history.table.SetRows(nil)
		g.history.sel = ""
		g.history.clear()
	}
}

// entryRows builds the file list's rows. A nil excluded means the list has
// no include column.
func entryRows(es []gitEntry, sel string, excluded map[string]bool) []ui.TableRow {
	rows := make([]ui.TableRow, len(es))
	for i, e := range es {
		file := e.change.Path
		if e.change.OrigPath != "" {
			file = e.change.OrigPath + " → " + e.change.Path
		}
		rows[i] = ui.TableRow{
			ID:       e.change.Path,
			Selected: e.change.Path == sel,
			Icon:     e.item.icon(),
			Cells: map[string]string{
				"change": changeLabel(e.change),
				"item":   e.item.label(),
				"type":   e.item.Type,
				"file":   file,
			},
		}
		if excluded != nil && !excluded[e.change.Path] {
			rows[i].Cells["inc"] = "1"
		}
	}
	return rows
}

// changeLabel names a change for the list. A renamed file that was also
// edited reads as renamed: the diff shows both.
func changeLabel(c gitx.Change) string {
	if c.Kind == gitx.Untracked {
		return "New"
	}
	return c.Kind.String()
}

func changeColor(_, colID, value string) []ui.TextSpan {
	if colID != "change" || value == "" {
		return nil
	}
	th := theme.Current()
	var c render.Color
	switch value {
	case "New", "Added":
		c = th.Success
	case "Deleted", "Conflicted":
		c = th.Error
	case "Renamed":
		c = th.Accent
	default:
		c = th.Warning
	}
	return []ui.TextSpan{{Start: 0, End: len(value), Color: c}}
}

func findEntry(es []gitEntry, path string) *gitEntry {
	for i := range es {
		if es[i].change.Path == path {
			return &es[i]
		}
	}
	return nil
}

func (v *gitDiffView) clear() {
	v.sel, v.key, v.diff, v.err, v.loading = "", "", nil, nil, false
}

// load fetches the diff for key and shows it when it arrives, unless the
// selection moved on.
func (g *Git) load(v *gitDiffView, key string, fetch func(ctx context.Context) ([]byte, error)) {
	v.key, v.loading, v.err = key, true, nil
	g.do(func(ctx context.Context) func() {
		data, err := fetch(ctx)
		return func() {
			if v.key != key {
				return
			}
			// A refresh that finds the same diff keeps the editor, and
			// with it the scroll position.
			same := err == nil && v.ed != nil && v.edSel == v.sel && bytes.Equal(data, v.diff)
			v.loading, v.diff, v.err = false, data, err
			if same {
				v.edKey = key
			}
		}
	})
}

func (g *Git) selectChange(path string) {
	e := findEntry(g.changes.entries, path)
	if e == nil || g.snap == nil || g.snap.repo == nil {
		return
	}
	g.changes.sel = path
	repo, c := g.snap.repo, e.change
	g.load(&g.changes, fmt.Sprintf("%d:%s", g.refreshSeq, path), func(ctx context.Context) ([]byte, error) {
		return repo.Diff(ctx, c)
	})
}

func (g *Git) commitByHash(hash string) *gitx.Commit {
	if g.snap == nil {
		return nil
	}
	for i := range g.snap.commits {
		if g.snap.commits[i].Hash == hash {
			return &g.snap.commits[i]
		}
	}
	return nil
}

func (g *Git) selectCommit(hash string) {
	c := g.commitByHash(hash)
	if c == nil || g.snap.repo == nil || hash == g.selCommit {
		return
	}
	g.selCommit = hash
	g.commitKey = hash
	g.commitBusy, g.commitErr = true, nil
	g.history.entries = nil
	g.history.table.SetRows(nil)
	g.history.clear()
	repo, commit := g.snap.repo, *c
	g.do(func(ctx context.Context) func() {
		changes, err := repo.CommitChanges(ctx, commit)
		var es []gitEntry
		for _, ch := range changes {
			rev, p := commit.Hash, ch.Path
			if ch.Kind == gitx.Deleted && len(commit.Parents) > 0 {
				rev = commit.Parents[0]
			}
			data, _ := repo.ShowFile(ctx, rev, p)
			es = append(es, gitEntry{change: ch, item: itemFor(p, data)})
		}
		sortEntries(es)
		return func() {
			if g.commitKey != hash {
				return
			}
			g.commitBusy, g.commitErr = false, err
			g.history.entries = es
			g.history.table.SetRows(entryRows(es, "", nil))
			if len(es) == 1 {
				g.selectCommitFile(es[0].change.Path)
			}
		}
	})
}

func (g *Git) selectCommitFile(path string) {
	e := findEntry(g.history.entries, path)
	c := g.commitByHash(g.selCommit)
	if e == nil || c == nil {
		return
	}
	g.history.sel = path
	repo, commit, ch := g.snap.repo, *c, e.change
	g.load(&g.history, commit.Hash+":"+path, func(ctx context.Context) ([]byte, error) {
		return repo.CommitDiff(ctx, commit, ch)
	})
}

// openItem opens the item a changed file holds.
func (g *Git) openItem(e *gitEntry) {
	if e == nil || e.item.ID == "" || g.deps.Open == nil {
		return
	}
	if !g.deps.Open(e.item.Kind, e.item.ID) {
		g.deps.Toast("This item is not in the workspace")
	}
}

// --- layout ---------------------------------------------------------------

// Poll applies the results of git commands that finished since the last
// frame. The app calls it every frame, whatever page shows: a clone
// finishes while the Spaces page is open.
func (g *Git) Poll(c *ui.Ctx) {
	if g.applyPending() {
		// The wake a result posts can land while the previous frame is
		// still running and be cleared when it presents. Ask for this
		// frame to be presented, so the result shows without waiting for
		// input.
		c.MarkNeedsPaint()
	}
}

func (g *Git) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		g.header(c),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(g.body(c)).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (g *Git) header(c *ui.Ctx) ui.View {
	th := c.Theme()
	views := []ui.View{ui.Title("Git")}
	if s := g.snap; s != nil && s.err == nil {
		views = append(views, g.branchInfo(c, s)...)
	}
	views = append(views, ui.Spacer())
	if g.busy != "" {
		views = append(views, ui.Caption(g.busy))
	}
	if s := g.snap; s != nil && s.err == nil {
		views = append(views, g.remoteActions(c, s)...)
	}
	refresh := ui.Button("git-refresh", ui.Text("Refresh")).IconStart(icons.RefreshCw).Ghost().HoverFill().
		Disabled(g.loading).OnClick(g.Refresh)
	views = append(views, refresh)
	return ui.Row(views...).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.M, th.Spacing.S)
}

func (g *Git) branchInfo(c *ui.Ctx, s *gitSnapshot) []ui.View {
	th := c.Theme()
	st := s.status
	branch := st.Branch
	if branch == "" {
		branch = "Detached at " + short(st.Head)
	}
	views := []ui.View{
		ui.Row(
			ui.Icon(icons.GitBranch, th.Metrics.IconSizeSM, th.ForegroundMuted),
			ui.Strong(branch),
		).Gap(th.Spacing.XS).Align(ui.AlignCenter),
	}
	var track string
	switch {
	case st.Head == "":
		track = "No commits yet"
	case st.Upstream == "":
		track = "Not tracking a remote branch"
	case st.Ahead == 0 && st.Behind == 0:
		track = "Up to date with " + st.Upstream
	case st.Behind == 0:
		track = fmt.Sprintf("%d ahead of %s", st.Ahead, st.Upstream)
	case st.Ahead == 0:
		track = fmt.Sprintf("%d behind %s", st.Behind, st.Upstream)
	default:
		track = fmt.Sprintf("%d ahead, %d behind %s", st.Ahead, st.Behind, st.Upstream)
	}
	views = append(views, ui.Caption(track))
	if !s.repo.Contained() {
		views = append(views, ui.Badge("Shared repository").
			Tooltip("The repository starts at "+s.repo.Root+", above this workspace. Only this workspace's files are shown."))
	}
	return views
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

func (g *Git) body(c *ui.Ctx) ui.View {
	th := c.Theme()
	s := g.snap
	switch {
	case s == nil:
		return emptyMessage(th, icons.GitBranch, "Reading the repository…", "")
	case errors.Is(s.err, gitx.ErrNotInstalled):
		return g.notInstalled(c)
	case errors.Is(s.err, gitx.ErrNotRepo):
		return gitEmpty(th, icons.GitBranch, "This workspace is not in a Git repository",
			"Set up a repository in its folder, "+s.dir+", to keep a history of its changes and share them "+
				"through a remote. Local secrets and cookies stay out of it.",
			ui.Button("git-init", ui.Text("Set up Git")).Primary().IconStart(icons.GitBranchPlus).
				Disabled(g.busy != "").OnClick(g.initRepo),
			ui.Button("git-check", ui.Text("Check again")).IconStart(icons.RefreshCw).
				Disabled(g.loading).OnClick(g.Refresh),
		)
	case s.err != nil:
		return gitEmpty(th, icons.TriangleAlert, "Git could not read the repository", s.err.Error(), g.checkAgain())
	}

	title := "Changes"
	if n := len(s.entries); n > 0 {
		title = fmt.Sprintf("Changes (%d)", n)
	}
	tabs := ui.Tabs("git-tabs", []ui.TabModel{{Title: title}, {Title: "History"}}).
		Selected(g.tab).Closable(false).
		OnSelectItem(func(i int, _ string) { g.tab = i }).
		TabBackground(th.Background)
	var page ui.View
	if g.tab == 1 {
		page = g.historyPane(c)
	} else {
		page = g.changesPane(c)
	}
	return ui.Column(
		tabs,
		ui.ViewOf(page).Grow(1),
	).Grow(1)
}

func (g *Git) checkAgain() ui.View {
	return ui.Button("git-check", ui.Text("Check again")).Primary().IconStart(icons.RefreshCw).
		Disabled(g.loading).OnClick(g.Refresh)
}

func (g *Git) notInstalled(c *ui.Ctx) ui.View {
	th := c.Theme()
	var hint string
	switch runtime.GOOS {
	case "darwin":
		hint = "Chapar uses the Git installed on this computer. Install the Xcode command line tools " +
			"by running xcode-select --install in Terminal, or install Git with Homebrew: brew install git."
	case "windows":
		hint = "Chapar uses the Git installed on this computer. Download Git for Windows and install it, then check again."
	default:
		hint = "Chapar uses the Git installed on this computer. Install it with your package manager, " +
			"for example sudo apt install git or sudo dnf install git, then check again."
	}
	return gitEmpty(th, icons.GitBranch, "Git is not installed", hint,
		ui.Button("git-download", ui.Text("Download Git")).IconStart(icons.Download).OnClick(func() {
			if err := langsrv.OpenURL("https://git-scm.com/downloads"); err != nil {
				g.deps.Error(err)
			}
		}),
		g.checkAgain(),
	)
}

// gitEmpty is an empty state with buttons under its text, the whole block
// centered in the page.
func gitEmpty(th *theme.Theme, icon icons.Icon, title, detail string, actions ...ui.View) ui.View {
	return ui.Column(
		ui.Row(ui.Icon(icon, th.Metrics.IconSizeMD*2, th.ForegroundMuted)).Justify(ui.JustifyCenter),
		ui.Row(ui.Subtitle(title)).Justify(ui.JustifyCenter),
		ui.Paragraph(detail).TextAlign(ui.AlignCenter).
			Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
		ui.Row(actions...).Gap(th.Spacing.S).Justify(ui.JustifyCenter).MarginTop(th.Spacing.S),
	).Gap(th.Spacing.M).PaddingXY(th.Spacing.XXL, th.Spacing.L).
		Justify(ui.JustifyCenter).Grow(1)
}

func (g *Git) changesPane(c *ui.Ctx) ui.View {
	th := c.Theme()
	if len(g.changes.entries) == 0 {
		detail := "Everything in this workspace matches the last commit."
		if g.snap.status.Head == "" {
			detail = "The repository has no commits, and this workspace has no files to commit."
		}
		return emptyMessage(th, icons.CircleCheck, "No changes", detail)
	}
	e := findEntry(g.changes.entries, g.changes.sel)
	list := ui.Column(
		g.commitBar(c),
		ui.ViewOf(g.changes.table).Grow(1),
	).Gap(th.Spacing.XS).Grow(1)
	return g.splitPane(c, "git-changes", list, g.diffPane(c, &g.changes, e, "Select a change to see its diff."))
}

func (g *Git) historyPane(c *ui.Ctx) ui.View {
	th := c.Theme()
	if len(g.snap.commits) == 0 {
		return emptyMessage(th, icons.GitCommitHorizontal, "No history yet", "Commits that change this workspace show here.")
	}
	var detail ui.View
	switch {
	case g.selCommit == "":
		detail = paneCard(th, container.MutedParagraph("Select a commit to see what it changed."))
	case g.commitBusy:
		detail = paneCard(th, container.MutedParagraph("Loading…"))
	case g.commitErr != nil:
		detail = paneCard(th, ui.Paragraph(g.commitErr.Error()).Style(ui.Spec{}.TextColor(ui.TokenError)))
	case len(g.history.entries) == 0:
		detail = paneCard(th, container.MutedParagraph("This commit changed no files in this workspace."))
	default:
		e := findEntry(g.history.entries, g.history.sel)
		detail = g.splitPane(c, "git-commit", ui.ViewOf(g.history.table).Grow(1), g.diffPane(c, &g.history, e, "Select a file to see its diff."))
	}
	return ui.Splitter("git-history-split", ui.Vertical, ui.ViewOf(g.commits).Grow(1), detail).
		Percents(35, 65).
		HandleOnHover().
		Grow(1)
}

// splitPane puts a file list next to, or above, its diff, the way the
// request page splits.
func (g *Git) splitPane(c *ui.Ctx, id string, list ui.View, diff ui.View) ui.View {
	dir := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)
	return ui.Splitter(id+"-split", dir, ui.ViewOf(list).Grow(1), diff).
		Percents(45, 55).
		HandleOnHover().
		Grow(1)
}

func (g *Git) diffPane(c *ui.Ctx, v *gitDiffView, e *gitEntry, empty string) ui.View {
	th := c.Theme()
	if e == nil {
		return paneCard(th, container.MutedParagraph(empty))
	}
	title := e.item.label()
	if e.item.Type != "File" {
		title = e.item.Type + " · " + title
	}
	var actions []ui.View
	if v == &g.changes {
		ent := *e
		if e.item.ID != "" && e.change.Kind != gitx.Deleted {
			actions = append(actions, ui.Button("git-open-item", ui.Text("Open")).IconStart(icons.SquareArrowOutUpRight).
				Ghost().HoverFill().OnClick(func() { g.openItem(&ent) }))
		}
		if e.change.Kind != gitx.Conflicted {
			actions = append(actions, ui.Button("git-discard", ui.Text("Discard")).IconStart(icons.Undo2).
				Ghost().HoverFill().Disabled(g.busy != "").
				Tooltip("Put this file back the way the last commit has it").
				OnClick(func() { g.confirmDiscard(ent) }))
		}
	}
	head := ui.Row(
		ui.Column(
			ui.Strong(title),
			ui.Caption(e.change.Path),
		).Gap(th.Spacing.XXS).Grow(1),
		ui.Row(actions...).Gap(th.Spacing.XS),
	).Gap(th.Spacing.S).Align(ui.AlignCenter)

	var body ui.View
	switch {
	case v.loading && (v.ed == nil || v.edSel != v.sel):
		body = container.MutedParagraph("Loading…")
	case v.err != nil:
		body = ui.Paragraph(v.err.Error()).Style(ui.Spec{}.TextColor(ui.TokenError))
	case !v.loading && len(v.diff) == 0:
		body = container.MutedParagraph("No content changes. Only the file's mode or its staged state differs.")
	default:
		g.syncEditor(v)
		body = container.ResponseEditorMenu(v.ed, c, g.deps.Container(), "changes.diff")
	}
	return paneCard(th, head, body)
}

// syncEditor rebuilds the diff editor when the diff changed.
func (g *Git) syncEditor(v *gitDiffView) {
	if v.loading || v.edKey == v.key && v.ed != nil {
		return
	}
	data := v.diff
	if len(data) > gitDiffLimit {
		data = append(data[:gitDiffLimit:gitDiffLimit], "\n… the diff is too long to show in full\n"...)
	}
	v.edKey, v.edSel = v.key, v.sel
	v.ed = container.ReplaceEditor(v.ed, data, &staticHighlighter{tokens: diffTokens(data)})
}
