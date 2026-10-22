package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/gitx"
	"github.com/chapar-rest/chapar/internal/yamlmerge"
	"github.com/chapar-rest/chapar/ui/container"
)

// --- branches -----------------------------------------------------------------

// branchPicker switches branches. Remote branches with no local branch
// are listed too; picking one makes a local branch that tracks it.
func (g *Git) branchPicker(c *ui.Ctx, s *gitSnapshot, current string) ui.View {
	var opts []ui.SelectOption
	sel := -1
	for i, b := range s.branches {
		label := b.Name
		if b.Remote {
			label += " (remote)"
		}
		opts = append(opts, ui.SelectOption{Label: label, Value: b.Name})
		if b.Current {
			sel = i
		}
	}
	if sel < 0 {
		// Detached HEAD, or a branch with no commits yet.
		opts = append([]ui.SelectOption{{Label: current, Value: ""}}, opts...)
		sel = 0
	}
	return ui.Select("git-branch", opts).Width(220).Selected(sel).
		Disabled(g.busy != "" || s.op != gitx.OpNone).
		OnChange(func(name string) {
			for _, b := range s.branches {
				if b.Name == name && !b.Current {
					g.switchBranch(b)
				}
			}
		})
}

// branchMenu holds the branch actions besides switching.
func (g *Git) branchMenu(s *gitSnapshot) ui.View {
	idle := g.busy == "" && s.op == gitx.OpNone
	cur := s.status.Branch
	items := []ui.MenuItem{
		{Label: "New branch…", Disabled: !idle || s.status.Head == "", OnSelect: g.askNewBranch},
	}
	if cur != "" {
		items = append(items, ui.MenuItem{Label: "Merge a branch into " + cur + "…",
			Disabled: !idle || len(s.branches) < 2, OnSelect: g.askMerge})
	}
	return ui.IconButton("git-branch-menu", icons.GitBranchPlus).Tooltip("Branch actions").Menu(items)
}

func (g *Git) switchBranch(b gitx.Branch) {
	if g.blockedByDirty("Switching branches") {
		return
	}
	repo := g.snap.repo
	g.action("Switching…", func(ctx context.Context) func() {
		err := repo.Switch(ctx, b)
		return func() {
			if err != nil {
				g.fail(err)
				return
			}
			g.reloadWorkspace()
			g.deps.Toast("Switched to " + strings.TrimPrefix(b.Name, remotePrefix(b)))
			g.Refresh()
		}
	})
}

// remotePrefix is "origin/" for a remote branch, else "".
func remotePrefix(b gitx.Branch) string {
	if !b.Remote {
		return ""
	}
	if i := strings.Index(b.Name, "/"); i >= 0 {
		return b.Name[:i+1]
	}
	return ""
}

func (g *Git) askNewBranch() {
	host := g.dialogs()
	if host == nil {
		return
	}
	repo := g.snap.repo
	host.ShowInput("New branch", "Branch name, such as fix/auth-headers", func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		g.action("Creating branch…", func(ctx context.Context) func() {
			err := repo.CreateBranch(ctx, name)
			return func() {
				if err != nil {
					g.fail(err)
					return
				}
				g.deps.Toast("Created and switched to " + name)
				g.Refresh()
			}
		})
	}, nil)
}

// askMerge picks a branch to merge into the current one.
func (g *Git) askMerge() {
	host := g.dialogs()
	if host == nil || g.blockedByDirty("Merging") {
		return
	}
	var opts []ui.SelectOption
	for _, b := range g.snap.branches {
		if !b.Current {
			opts = append(opts, ui.SelectOption{Label: b.Name, Value: b.Name})
		}
	}
	if len(opts) == 0 {
		return
	}
	pick := opts[0].Value
	cur := g.snap.status.Branch
	host.Show(ui.DialogOpts{
		Title:  "Merge into " + cur,
		Width:  440,
		Height: 260,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			sel := 0
			for i, o := range opts {
				if o.Value == pick {
					sel = i
				}
			}
			return ui.Column(
				ui.Paragraph("Bring another branch's commits into "+cur+". Fields both branches changed "+
					"differently are listed for you to choose from."),
				ui.Select("git-merge-pick", opts).Selected(sel).OnChange(func(v string) { pick = v }),
			).Gap(th.Spacing.M).Padding(th.Spacing.L)
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel"},
			{Label: "Merge", Primary: true, OnClick: func() { g.merge(pick) }},
		},
	})
}

// merge merges ref into the current branch. Conflicts leave the merge in
// progress, with the Changes tab listing them.
func (g *Git) merge(ref string) {
	if g.blockedByDirty("Merging") {
		return
	}
	repo := g.snap.repo
	g.action("Merging…", func(ctx context.Context) func() {
		conflicts, err := repo.Merge(ctx, ref)
		return func() { g.afterStop(err, conflicts, "Merged "+ref) }
	})
}

// afterStop reports a merge or revert: done, stopped with conflicts, or
// failed.
func (g *Git) afterStop(err error, conflicts bool, done string) {
	if err != nil {
		g.fail(err)
		return
	}
	g.reloadWorkspace()
	if conflicts {
		g.tab = 0
		g.deps.Toast("Some fields changed on both sides. Pick a version for each, then finish.")
	} else {
		g.deps.Toast(done)
	}
	g.Refresh()
}

// offerMerge follows a pull that couldn't fast-forward.
func (g *Git) offerMerge() {
	host := g.dialogs()
	up := g.snap.status.Upstream
	if host == nil || up == "" {
		g.fail(gitx.ErrDiverged)
		return
	}
	host.ShowAction("Merge the remote's changes?",
		"This branch and "+up+" both have new commits. Chapar can merge them, field by field, and lists "+
			"any field both sides changed differently for you to choose.",
		func() { g.merge(up) }, nil)
}

// --- in progress ---------------------------------------------------------

func (g *Git) conflicts() int {
	n := 0
	for _, e := range g.changes.entries {
		if e.change.Kind == gitx.Conflicted {
			n++
		}
	}
	return n
}

// opBanner replaces the commit box while git is in the middle of a merge
// or revert.
func (g *Git) opBanner(c *ui.Ctx) ui.View {
	th := c.Theme()
	s := g.snap
	if s.op != gitx.OpMerge && s.op != gitx.OpRevert {
		return ui.Row(
			ui.Icon(icons.TriangleAlert, th.Metrics.IconSizeSM, th.Warning),
			ui.Paragraph("A "+s.op.String()+" is in progress. Finish or abort it in a terminal, then refresh.").Grow(1),
		).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.M, th.Spacing.S)
	}
	title := s.opMsg
	if title == "" {
		title = "A " + s.op.String() + " is in progress"
	}
	n := g.conflicts()
	status := "Every conflict is resolved"
	switch {
	case n == 1:
		status = "1 file has conflicts"
	case n > 1:
		status = fmt.Sprintf("%d files have conflicts", n)
	}
	finish := "Finish merge"
	if s.op == gitx.OpRevert {
		finish = "Finish revert"
	}
	idle := g.busy == ""
	return ui.Row(
		ui.Icon(icons.GitMerge, th.Metrics.IconSizeSM, th.Accent),
		ui.Column(
			ui.Strong(title),
			ui.Caption(status),
		).Gap(th.Spacing.XXS).Grow(1),
		ui.Button("git-abort", ui.Text("Abort")).Disabled(!idle).OnClick(g.confirmAbort),
		ui.Button("git-finish", ui.Text(finish)).Primary().Disabled(!idle || n > 0).OnClick(g.finish),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.M, th.Spacing.S)
}

func (g *Git) confirmAbort() {
	host := g.dialogs()
	if host == nil || g.blockedByDirty("Aborting") {
		return
	}
	repo, op := g.snap.repo, g.snap.op
	host.ShowAction("Abort the "+op.String()+"?",
		"The branch and its files go back to how they were before it started, and the versions you picked are lost.",
		func() {
			g.action("Aborting…", func(ctx context.Context) func() {
				err := repo.Abort(ctx, op)
				return func() {
					if err != nil {
						g.fail(err)
						return
					}
					g.reloadWorkspace()
					g.Refresh()
				}
			})
		}, nil)
}

func (g *Git) finish() {
	repo, op := g.snap.repo, g.snap.op
	g.action("Committing…", func(ctx context.Context) func() {
		err := repo.Continue(ctx, op)
		return func() {
			if err != nil {
				g.fail(err)
				return
			}
			g.deps.Toast("Finished the " + op.String())
			g.Refresh()
		}
	})
}

// --- resolving a conflicted file -------------------------------------------

// gitResolve is a conflicted file, merged field by field, waiting for a
// version to be picked for each field both sides changed.
type gitResolve struct {
	path string
	key  string
	// loading is set while the versions are read.
	loading bool
	err     error
	merged  *yamlmerge.Result
	// whole is why the file can only be taken whole from one side, or "".
	whole    string
	versions gitx.Versions
	sides    []yamlmerge.Side
}

func (g *Git) loadResolve(path string) {
	key := fmt.Sprintf("%d:%s", g.refreshSeq, path)
	if g.resolve.key == key {
		return
	}
	g.resolve = gitResolve{path: path, key: key, loading: true}
	repo := g.snap.repo
	g.do(func(ctx context.Context) func() {
		v, err := repo.ConflictVersions(ctx, path)
		var merged *yamlmerge.Result
		whole := ""
		if err == nil {
			if !strings.HasSuffix(path, ".yaml") {
				whole = "This isn't a Chapar file, so Chapar can't merge it field by field."
			} else if m, merr := yamlmerge.Merge(v.Base, v.Ours, v.Theirs); merr != nil {
				whole = "Chapar can't merge this file field by field: " + merr.Error()
			} else {
				merged = m
			}
		}
		return func() {
			if g.resolve.key != key {
				return
			}
			g.resolve.loading, g.resolve.err = false, err
			g.resolve.versions, g.resolve.merged, g.resolve.whole = v, merged, whole
			if merged != nil {
				g.resolve.sides = make([]yamlmerge.Side, len(merged.Conflicts))
			}
		}
	})
}

// sideNames name the two sides for the current operation.
func (g *Git) sideNames() (ours, theirs string) {
	ours = "This branch"
	if b := g.snap.status.Branch; b != "" {
		ours = b
	}
	if g.snap.op == gitx.OpRevert {
		return ours, "The revert"
	}
	return ours, "Incoming"
}

func (g *Git) resolvePane(c *ui.Ctx, e *gitEntry) ui.View {
	th := c.Theme()
	r := &g.resolve
	head := ui.Column(
		ui.Strong(e.item.Type+" · "+e.item.label()),
		ui.Caption(e.change.Path),
	).Gap(th.Spacing.XXS)
	switch {
	case r.path != e.change.Path || r.loading:
		return paneCard(th, head, container.MutedParagraph("Reading both versions…"))
	case r.err != nil:
		return paneCard(th, head, ui.Paragraph(r.err.Error()).Style(ui.Spec{}.TextColor(ui.TokenError)))
	case r.merged == nil:
		return paneCard(th, head, g.wholeFileChoice(c))
	}
	ours, theirs := g.sideNames()
	n := len(r.merged.Conflicts)
	var intro string
	switch n {
	case 0:
		intro = "Both sides changed this file, in different fields. Chapar merged them; save to mark it resolved."
	case 1:
		intro = "Both sides changed one field differently. Pick the version to keep; every other change is merged."
	default:
		intro = fmt.Sprintf("Both sides changed %d fields differently. Pick the version to keep for each; "+
			"every other change is merged.", n)
	}
	rows := []ui.View{}
	for i, cf := range r.merged.Conflicts {
		i := i
		path := cf.Path
		if path == "" {
			path = "The whole file"
		}
		rows = append(rows, ui.Column(
			ui.Strong(path),
			g.choice(c, fmt.Sprintf("git-res-%d-o", i), ours, cf.Ours, r.sides[i] == yamlmerge.Ours,
				func() { r.sides[i] = yamlmerge.Ours }),
			g.choice(c, fmt.Sprintf("git-res-%d-t", i), theirs, cf.Theirs, r.sides[i] == yamlmerge.Theirs,
				func() { r.sides[i] = yamlmerge.Theirs }),
		).Gap(th.Spacing.XS).PaddingBottom(th.Spacing.M))
	}
	picked := true
	for _, s := range r.sides {
		picked = picked && s != yamlmerge.Unset
	}
	all := func(side yamlmerge.Side) func() {
		return func() {
			for i := range r.sides {
				r.sides[i] = side
			}
		}
	}
	actions := []ui.View{}
	if n > 1 {
		actions = append(actions,
			ui.Button("git-res-all-o", ui.Text("All from "+ours)).Ghost().HoverFill().OnClick(all(yamlmerge.Ours)),
			ui.Button("git-res-all-t", ui.Text("All from "+strings.ToLower(theirs))).Ghost().HoverFill().OnClick(all(yamlmerge.Theirs)),
		)
	}
	save := ui.Button("git-res-save", ui.Text("Mark resolved")).Primary().
		Disabled(!picked || g.busy != "").OnClick(g.saveResolution)
	if !picked {
		save = save.Tooltip("Pick a version for every field first")
	}
	actions = append(actions, ui.Spacer(), save)
	return paneCard(th,
		head,
		ui.Paragraph(intro),
		ui.Scroll("git-res-scroll", ui.Column(rows...).Gap(th.Spacing.S)).Grow(1),
		ui.Row(actions...).Gap(th.Spacing.S).Align(ui.AlignCenter),
	)
}

// choice is one side's value for a conflicted field.
func (g *Git) choice(c *ui.Ctx, id, side string, v any, on bool, pick func()) ui.View {
	th := c.Theme()
	val := yamlmerge.Show(v)
	var body ui.View
	if yamlmerge.IsMissing(v) {
		body = container.MutedParagraph("Removed")
	} else if val == "" {
		body = container.MutedParagraph("Empty")
	} else {
		body = ui.Paragraph(val).Style(ui.Spec{}.TextColor(ui.TokenForeground))
	}
	row := ui.Column(
		ui.Radio(id, side).Check(on).OnClick(pick),
		ui.Column(body).PaddingLeft(th.Spacing.XL),
	).Gap(th.Spacing.XXS).Padding(th.Spacing.S).Radius(th.Radius.Medium).
		Border(ui.TokenBorder, th.Stroke.Thin)
	if on {
		row = row.Background(ui.TokenListActive)
	}
	return row
}

// wholeFileChoice settles a file that can't be merged field by field by
// taking one side's version of it.
func (g *Git) wholeFileChoice(c *ui.Ctx) ui.View {
	th := c.Theme()
	r := &g.resolve
	ours, theirs := g.sideNames()
	label := func(name string, data []byte) string {
		if data == nil {
			return "Delete it, as " + name + " did"
		}
		return "Keep " + name + "'s version"
	}
	return ui.Column(
		container.MutedParagraph(r.whole),
		ui.Row(
			ui.Button("git-res-whole-o", ui.Text(label(ours, r.versions.Ours))).
				Disabled(g.busy != "").OnClick(func() { g.resolveWith(r.path, r.versions.Ours) }),
			ui.Button("git-res-whole-t", ui.Text(label(strings.ToLower(theirs), r.versions.Theirs))).
				Disabled(g.busy != "").OnClick(func() { g.resolveWith(r.path, r.versions.Theirs) }),
		).Gap(th.Spacing.S),
	).Gap(th.Spacing.M)
}

func (g *Git) saveResolution() {
	r := &g.resolve
	data, err := r.merged.Apply(r.sides)
	if err != nil {
		g.deps.Error(err)
		return
	}
	g.resolveWith(r.path, data)
}

// resolveWith writes the resolved file, or deletes it for nil data, and
// marks it resolved.
func (g *Git) resolveWith(path string, data []byte) {
	if g.blockedByDirty("Resolving") {
		return
	}
	repo := g.snap.repo
	g.action("Resolving…", func(ctx context.Context) func() {
		err := repo.Resolve(ctx, path, data)
		return func() {
			if err != nil {
				g.fail(err)
				return
			}
			g.reloadWorkspace()
			g.Refresh()
		}
	})
}

// --- revert ----------------------------------------------------------------

// commitHeader names the selected commit above its files, with Revert.
func (g *Git) commitHeader(c *ui.Ctx) ui.View {
	th := c.Theme()
	cm := g.commitByHash(g.selCommit)
	if cm == nil {
		return nil
	}
	commit := *cm
	idle := g.busy == "" && g.snap.op == gitx.OpNone
	return ui.Row(
		ui.Column(
			ui.Strong(commit.Subject),
			ui.Caption(commit.Short+" · "+commit.Author+" · "+commit.When.Format("2006-01-02 15:04")),
		).Gap(th.Spacing.XXS).Grow(1),
		ui.Button("git-revert", ui.Text("Revert")).IconStart(icons.Undo2).Ghost().HoverFill().
			Disabled(!idle).Tooltip("Make a new commit that undoes this one").
			OnClick(func() { g.confirmRevert(commit) }),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.S, th.Spacing.XS)
}

func (g *Git) confirmRevert(cm gitx.Commit) {
	host := g.dialogs()
	if host == nil || g.blockedByDirty("Reverting") {
		return
	}
	host.ShowAction("Revert “"+cm.Subject+"”?",
		"Chapar makes a new commit that undoes its changes. The commit stays in the history.",
		func() { g.revert(cm) }, nil)
}

func (g *Git) revert(cm gitx.Commit) {
	repo := g.snap.repo
	g.action("Reverting…", func(ctx context.Context) func() {
		conflicts, err := repo.Revert(ctx, cm)
		return func() { g.afterStop(err, conflicts, "Reverted "+cm.Short) }
	})
}
