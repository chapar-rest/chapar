package pages

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/gitx"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/secret"
	"github.com/chapar-rest/chapar/ui/container"
)

// gitForm holds what the git dialogs' fields hold while one is open.
type gitForm struct {
	url, username, token string
	name, email          string
	problem              string
}

// action runs work off the UI goroutine with the page marked busy, so
// the action buttons stay disabled until it is done.
func (g *Git) action(label string, work func(ctx context.Context) func()) {
	g.busy = label
	g.doFor(gitActionTimeout, func(ctx context.Context) func() {
		apply := work(ctx)
		return func() {
			g.busy = ""
			apply()
		}
	})
}

// fail shows err the way a user can act on it, then reads the repository
// again, since a failed command may have changed it part way.
func (g *Git) fail(err error) {
	g.deps.Error(gitError(err, g.snap))
	g.Refresh()
}

// gitMessage is an error worded for the user, in full sentences.
type gitMessage string

func (m gitMessage) Error() string { return string(m) }

// gitError rewords the errors the user can fix.
func gitError(err error, s *gitSnapshot) error {
	switch {
	case errors.Is(err, gitx.ErrDiverged):
		return gitMessage("This branch and the remote both have new commits. Chapar can't combine them yet: " +
			"merge or rebase in a terminal, then refresh.")
	case errors.Is(err, gitx.ErrRejected):
		return gitMessage("The remote has commits this branch doesn't have. Pull them first, then push.")
	case errors.Is(err, gitx.ErrNoUpstream):
		branch := "this branch"
		if s != nil && s.status != nil && s.status.Branch != "" {
			branch = s.status.Branch
		}
		return gitMessage("The remote has no branch named " + branch + " yet. Push to create it.")
	}
	return err
}

func (g *Git) dialogs() *ui.DialogHost {
	if g.deps.Dialogs == nil {
		return nil
	}
	return g.deps.Dialogs()
}

// blockedByDirty reports, and explains, that an action which rewrites
// workspace files has to wait for unsaved tabs.
func (g *Git) blockedByDirty(what string) bool {
	if g.deps.HasDirty == nil || !g.deps.HasDirty() {
		return false
	}
	g.deps.Error(gitMessage("Save or close the tabs with unsaved changes first. " + what +
		" replaces workspace files, and those tabs would still hold the old versions."))
	return true
}

// --- set up ---------------------------------------------------------------

// initRepo makes the workspace folder a repository, with a .gitignore
// that keeps machine-local state out of it.
func (g *Git) initRepo() {
	dir := g.snap.dir
	g.action("Setting up…", func(ctx context.Context) func() {
		git, err := gitx.Find(ctx)
		if err == nil {
			err = git.Init(ctx, dir)
		}
		if err == nil {
			err = repository.EnsureGitignore(dir)
		}
		return func() {
			if err != nil {
				g.fail(err)
				return
			}
			g.deps.Toast("Git is set up. Commit the workspace to start its history.")
			g.Refresh()
		}
	})
}

// --- commit ---------------------------------------------------------------

func (g *Git) setIncluded(path string, on bool) {
	if on {
		delete(g.excluded, path)
	} else {
		g.excluded[path] = true
	}
}

func (g *Git) includeAll(on bool) {
	for _, e := range g.changes.entries {
		g.setIncluded(e.change.Path, on)
	}
	g.changes.table.SetRows(entryRows(g.changes.entries, g.changes.sel, g.excluded))
}

func (g *Git) included() []gitx.Change {
	var out []gitx.Change
	for _, e := range g.changes.entries {
		if !g.excluded[e.change.Path] {
			out = append(out, e.change)
		}
	}
	return out
}

func (g *Git) commitBar(c *ui.Ctx) ui.View {
	th := c.Theme()
	n := len(g.included())
	label := "Commit"
	if n > 0 {
		label = fmt.Sprintf("Commit %d", n)
	}
	ready := n > 0 && strings.TrimSpace(g.message) != "" && g.busy == ""
	var tip string
	switch {
	case n == 0:
		tip = "Tick the changes to commit"
	case strings.TrimSpace(g.message) == "":
		tip = "Write a commit message"
	}
	commit := ui.Button("git-commit", ui.Text(label)).Primary().IconStart(icons.GitCommitHorizontal).
		Disabled(!ready).OnClick(g.commit)
	if tip != "" {
		commit = commit.Tooltip(tip)
	}
	var note ui.View
	if g.deps.HasDirty != nil && g.deps.HasDirty() {
		note = ui.Caption("Unsaved tabs are not in the commit until you save them").
			Style(ui.Spec{}.TextColor(ui.TokenWarning))
	}
	return ui.Column(
		ui.Row(
			ui.TextField("git-message", g.message).Placeholder("Commit message").
				OnChange(func(s string) { g.message = s }).
				OnSubmit(func(string) {
					if ready {
						g.commit()
					}
				}).Grow(1),
			commit,
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
		ui.Row(
			ui.Button("git-all", ui.Text("Select all")).Ghost().HoverFill().OnClick(func() { g.includeAll(true) }),
			ui.Button("git-none", ui.Text("Select none")).Ghost().HoverFill().OnClick(func() { g.includeAll(false) }),
			ui.Spacer(),
			note,
		).Gap(th.Spacing.XXS).Align(ui.AlignCenter),
	).Gap(th.Spacing.XS).PaddingXY(th.Spacing.S, th.Spacing.XS)
}

// commit records the ticked changes. When git has no name and email to
// commit with, it asks for them first.
func (g *Git) commit() {
	changes := g.included()
	msg := strings.TrimSpace(g.message)
	if len(changes) == 0 || msg == "" || g.snap == nil || g.snap.repo == nil {
		return
	}
	repo := g.snap.repo
	g.action("Committing…", func(ctx context.Context) func() {
		if name, email := repo.Identity(ctx); name == "" || email == "" {
			return func() { g.askIdentity(name, email, g.commit) }
		}
		_, err := repo.Commit(ctx, msg, changes)
		return func() {
			if err != nil {
				g.fail(err)
				return
			}
			g.message = ""
			if len(changes) == 1 {
				g.deps.Toast("Committed 1 change")
			} else {
				g.deps.Toast(fmt.Sprintf("Committed %d changes", len(changes)))
			}
			g.Refresh()
		}
	})
}

// askIdentity asks for the name and email commits are signed with, sets
// them for this repository, then runs next.
func (g *Git) askIdentity(name, email string, next func()) {
	host := g.dialogs()
	if host == nil {
		return
	}
	if g.form.problem == "" {
		g.form = gitForm{name: name, email: email}
	}
	repo := g.snap.repo
	host.Show(ui.DialogOpts{
		Title:  "Who is committing?",
		Width:  440,
		Height: 330,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.Paragraph("Git records a name and email with every commit. Chapar sets them for this repository only."),
				g.formProblem(),
				fieldLabel("Name"),
				ui.TextField("git-id-name", g.form.name).Placeholder("Ada Lovelace").
					OnChange(func(s string) { g.form.name = s }),
				fieldLabel("Email"),
				ui.TextField("git-id-email", g.form.email).Placeholder("ada@example.com").
					OnChange(func(s string) { g.form.email = s }),
			).Gap(th.Spacing.S).Padding(th.Spacing.L)
		},
		OnDismiss: func() { g.form = gitForm{} },
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { g.form = gitForm{} }},
			{Label: "Save and commit", Primary: true, OnClick: func() {
				n, e := strings.TrimSpace(g.form.name), strings.TrimSpace(g.form.email)
				if n == "" || !strings.Contains(e, "@") {
					g.form.problem = "Enter a name and an email address."
					// Dialog actions always dismiss; show it again.
					g.askIdentity(n, e, next)
					return
				}
				g.form = gitForm{}
				g.action("Committing…", func(ctx context.Context) func() {
					err := repo.SetIdentity(ctx, n, e)
					return func() {
						if err != nil {
							g.fail(err)
							return
						}
						next()
					}
				})
			}},
		},
	})
}

func fieldLabel(s string) ui.View {
	return ui.Text(s).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
}

// formProblem shows why the last try at a dialog didn't go through.
func (g *Git) formProblem() ui.View {
	if g.form.problem == "" {
		return nil
	}
	return ui.Paragraph(g.form.problem).Style(ui.Spec{}.TextColor(ui.TokenError))
}

// --- discard --------------------------------------------------------------

func (g *Git) confirmDiscard(e gitEntry) {
	if g.blockedByDirty("Discarding") {
		return
	}
	host := g.dialogs()
	if host == nil {
		return
	}
	var what string
	switch e.change.Kind {
	case gitx.Untracked, gitx.Added:
		what = fmt.Sprintf("%s is new since the last commit, so discarding deletes it.", e.change.Path)
	case gitx.Deleted:
		what = fmt.Sprintf("%s comes back the way the last commit has it.", e.change.Path)
	case gitx.Renamed:
		what = fmt.Sprintf("%s goes back to its old name, with the content the last commit has.", e.change.Path)
	default:
		what = fmt.Sprintf("Your changes to %s are lost.", e.change.Path)
	}
	repo := g.snap.repo
	host.ShowAction("Discard changes to "+e.item.label()+"?", what+" This can't be undone.", func() {
		g.action("Discarding…", func(ctx context.Context) func() {
			err := repo.Discard(ctx, e.change)
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

func (g *Git) reloadWorkspace() {
	if g.deps.Reload != nil {
		g.deps.Reload()
	}
}

// --- remote ---------------------------------------------------------------

// remoteActions are the header's buttons for the remote.
func (g *Git) remoteActions(c *ui.Ctx, s *gitSnapshot) []ui.View {
	idle := g.busy == ""
	if s.remote == "" {
		return []ui.View{
			ui.Button("git-connect", ui.Text("Connect remote")).IconStart(icons.Link).
				Disabled(!idle).OnClick(func() { g.showRemote("") }),
		}
	}
	st := s.status
	pull := "Pull"
	if st.Behind > 0 {
		pull = fmt.Sprintf("Pull %d", st.Behind)
	}
	push := "Push"
	if st.Ahead > 0 {
		push = fmt.Sprintf("Push %d", st.Ahead)
	}
	branch := st.Branch != ""
	items := []ui.MenuItem{
		{Label: "Fetch", Disabled: !idle, OnSelect: g.fetch},
		{Label: "Change remote…", Disabled: !idle, OnSelect: func() { g.showRemote(s.remoteURL) }},
	}
	if gitx.IsHTTP(s.remoteURL) {
		remote := s.remoteURL
		items = append(items, ui.MenuItem{Label: "Forget saved token", OnSelect: func() {
			if err := gitx.ForgetAuth(remote); err != nil {
				g.deps.Error(err)
				return
			}
			g.deps.Toast("Forgot the token for " + hostOf(remote))
		}})
	}
	return []ui.View{
		ui.Button("git-pull", ui.Text(pull)).IconStart(icons.ArrowDown).Ghost().HoverFill().
			Disabled(!idle || !branch).Tooltip("Bring in new commits from " + s.remote).OnClick(g.pull),
		ui.Button("git-push", ui.Text(push)).IconStart(icons.ArrowUp).Ghost().HoverFill().
			Disabled(!idle || !branch || st.Head == "").Tooltip("Send this branch's commits to " + s.remote).OnClick(g.push),
		ui.IconButton("git-remote-menu", icons.Ellipsis).Tooltip("More").Menu(items),
	}
}

func hostOf(remote string) string {
	if u, err := url.Parse(remote); err == nil && u.Host != "" {
		return u.Host
	}
	return remote
}

// network runs op against the remote with the token saved for its server.
// When the remote turns the credentials down, it asks for new ones and
// tries again.
func (g *Git) network(label string, op func(ctx context.Context, auth gitx.Auth) error, done func()) {
	remote := g.snap.remoteURL
	g.action(label, func(ctx context.Context) func() {
		auth := gitx.LoadAuth(remote)
		err := op(ctx, auth)
		return func() {
			if errors.Is(err, gitx.ErrAuth) && gitx.IsHTTP(remote) {
				g.askToken(remote, auth.Username, func() { g.network(label, op, done) })
				return
			}
			if errors.Is(err, gitx.ErrDiverged) {
				g.offerMerge()
				return
			}
			if err != nil {
				g.fail(err)
				return
			}
			done()
		}
	})
}

func (g *Git) fetch() {
	repo := g.snap.repo
	g.network("Fetching…", func(ctx context.Context, auth gitx.Auth) error {
		return repo.Fetch(ctx, auth)
	}, g.Refresh)
}

func (g *Git) push() {
	repo, st, remote := g.snap.repo, g.snap.status, g.snap.remote
	g.network("Pushing…", func(ctx context.Context, auth gitx.Auth) error {
		return repo.Push(ctx, auth, st)
	}, func() {
		g.deps.Toast("Pushed " + st.Branch + " to " + remote)
		g.Refresh()
	})
}

func (g *Git) pull() {
	if g.blockedByDirty("Pulling") {
		return
	}
	repo, st := g.snap.repo, g.snap.status
	var moved bool
	g.network("Pulling…", func(ctx context.Context, auth gitx.Auth) error {
		var err error
		moved, err = repo.Pull(ctx, auth, st)
		return err
	}, func() {
		if moved {
			g.reloadWorkspace()
			g.deps.Toast("Pulled the latest changes")
		} else {
			g.deps.Toast("Already up to date")
		}
		g.Refresh()
	})
}

// showRemote asks for the remote's URL, and for an HTTPS remote,
// optionally a user name and token.
func (g *Git) showRemote(current string) {
	host := g.dialogs()
	if host == nil {
		return
	}
	if g.form.problem == "" {
		g.form = gitForm{url: current}
		if current != "" {
			g.form.username = gitx.LoadAuth(current).Username
		}
	}
	repo := g.snap.repo
	title := "Connect a remote"
	if current != "" {
		title = "Change the remote"
	}
	host.Show(ui.DialogOpts{
		Title:  title,
		Width:  520,
		Height: 430,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			views := []ui.View{
				ui.Paragraph("The repository to push this workspace to and pull it from, such as " +
					"https://github.com/team/api.git or git@github.com:team/api.git."),
				g.formProblem(),
				fieldLabel("URL"),
				ui.TextField("git-remote-url", g.form.url).Placeholder("https://github.com/team/api.git").
					OnChange(func(s string) { g.form.url = strings.TrimSpace(s) }),
			}
			if gitx.IsHTTP(g.form.url) {
				views = append(views, g.tokenFields(c)...)
			} else if g.form.url != "" {
				views = append(views, container.MutedParagraph(
					"Chapar connects over SSH with your SSH keys and agent, the same way git does in a terminal."))
			}
			return ui.Column(views...).Gap(th.Spacing.S).Padding(th.Spacing.L)
		},
		OnDismiss: func() { g.form = gitForm{} },
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { g.form = gitForm{} }},
			{Label: "Save", Primary: true, OnClick: func() {
				f := g.form
				if f.url == "" {
					g.form.problem = "Enter the remote's URL."
					g.showRemote(current)
					return
				}
				g.form = gitForm{}
				g.saveRemote(repo, f)
			}},
		},
	})
}

// saveRemote points the repository at f.url and saves f's token.
func (g *Git) saveRemote(repo *gitx.Repo, f gitForm) {
	g.action("Connecting…", func(ctx context.Context) func() {
		err := repo.SetRemote(ctx, f.url)
		if err == nil && f.token != "" {
			err = gitx.SaveAuth(f.url, gitx.Auth{Username: f.username, Password: f.token})
		}
		return func() {
			if err != nil {
				g.fail(err)
				return
			}
			g.deps.Toast("Remote saved")
			g.Refresh()
		}
	})
}

// tokenFields are the user name and token fields for an HTTPS remote.
func (g *Git) tokenFields(c *ui.Ctx) []ui.View {
	return []ui.View{
		fieldLabel("User name"),
		ui.TextField("git-remote-user", g.form.username).Placeholder("Your user name on the server").
			OnChange(func(s string) { g.form.username = strings.TrimSpace(s) }),
		fieldLabel("Access token"),
		ui.TextField("git-remote-token", g.form.token).Password(true).
			Placeholder("A personal access token with repository access").
			OnChange(func(s string) { g.form.token = strings.TrimSpace(s) }),
		container.MutedParagraph("Leave these empty to use the credentials git already has. A token you enter is " +
			"saved in " + secretStoreName() + " and used for every repository on this server."),
	}
}

// askToken asks for a user name and token after the server refused the
// ones git had, saves them, then runs retry.
func (g *Git) askToken(remote, username string, retry func()) {
	host := g.dialogs()
	if host == nil {
		return
	}
	if g.form.problem == "" {
		g.form = gitForm{url: remote, username: username}
	}
	host.Show(ui.DialogOpts{
		Title:  "Sign in to " + hostOf(remote),
		Width:  480,
		Height: 400,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			views := []ui.View{
				ui.Paragraph(hostOf(remote) + " needs a user name and an access token to reach this repository. " +
					"Most servers no longer take an account password here."),
				g.formProblem(),
			}
			views = append(views, g.tokenFields(c)[:4]...)
			views = append(views, container.MutedParagraph("Chapar saves the token in "+secretStoreName()+
				" and uses it for every repository on this server."))
			return ui.Column(views...).Gap(th.Spacing.S).Padding(th.Spacing.L)
		},
		OnDismiss: func() { g.form = gitForm{} },
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { g.form = gitForm{} }},
			{Label: "Sign in", Primary: true, OnClick: func() {
				f := g.form
				if f.username == "" || f.token == "" {
					g.form.problem = "Enter a user name and a token."
					g.askToken(remote, f.username, retry)
					return
				}
				g.form = gitForm{}
				if err := gitx.SaveAuth(remote, gitx.Auth{Username: f.username, Password: f.token}); err != nil {
					g.deps.Error(fmt.Errorf("couldn't save the token: %w", err))
					return
				}
				retry()
			}},
		},
	})
}

// secretStoreName is what the platform calls its secret store.
func secretStoreName() string { return secret.NewOSStore().Name() }

// --- clone ------------------------------------------------------------------

// ShowClone asks for a repository and clones it as a new workspace.
func (g *Git) ShowClone() {
	host := g.dialogs()
	if host == nil {
		return
	}
	host.Show(ui.DialogOpts{
		Title:  "Clone a space",
		Width:  520,
		Height: 430,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			views := []ui.View{
				ui.Paragraph("Clone a Git repository that holds a Chapar workspace, such as one a teammate pushed. " +
					"Secret values aren't in the repository: set them on this machine after cloning."),
				g.formProblem(),
				fieldLabel("Repository URL"),
				ui.TextField("git-clone-url", g.form.url).Placeholder("https://github.com/team/api.git").
					OnChange(func(s string) { g.form.url = strings.TrimSpace(s) }),
			}
			if gitx.IsHTTP(g.form.url) {
				views = append(views, g.tokenFields(c)...)
			}
			return ui.Column(views...).Gap(th.Spacing.S).Padding(th.Spacing.L)
		},
		OnDismiss: func() { g.form = gitForm{} },
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { g.form = gitForm{} }},
			{Label: "Clone", Primary: true, OnClick: func() {
				f := g.form
				if f.url == "" {
					g.form.problem = "Enter the repository's URL."
					g.ShowClone()
					return
				}
				g.form = gitForm{}
				g.clone(f)
			}},
		},
	})
}

// clone runs git clone into a hidden folder next to the workspaces, then
// moves it to the folder its workspace is named after.
func (g *Git) clone(f gitForm) {
	dataDir := ""
	if g.deps.DataDir != nil {
		dataDir = g.deps.DataDir()
	}
	var existing []*domain.Workspace
	if g.deps.Workspaces != nil {
		existing = g.deps.Workspaces()
	}
	g.deps.Toast("Cloning " + f.url + "…")
	g.action("Cloning…", func(ctx context.Context) func() {
		ws, err := cloneWorkspace(ctx, dataDir, f, existing)
		return func() {
			if errors.Is(err, gitx.ErrAuth) {
				// Ask again, with the reason and what was typed.
				g.form = f
				g.form.problem = hostOf(f.url) + " turned the credentials down. Enter a user name and an access token."
				if !gitx.IsHTTP(f.url) {
					g.form.problem = err.Error()
				}
				g.ShowClone()
				return
			}
			if ws == nil {
				g.deps.Error(fmt.Errorf("couldn't clone %s: %w", f.url, err))
				return
			}
			if err != nil {
				g.deps.Error(err)
			}
			g.deps.Toast("Cloned " + ws.GetName())
			if g.deps.Cloned != nil {
				g.deps.Cloned(ws)
			}
		}
	})
}

func cloneWorkspace(ctx context.Context, dataDir string, f gitForm, existing []*domain.Workspace) (*domain.Workspace, error) {
	git, err := gitx.Find(ctx)
	if err != nil {
		return nil, err
	}
	auth := gitx.Auth{Username: f.username, Password: f.token}
	if f.token == "" {
		auth = gitx.LoadAuth(f.url)
	}
	tmp, err := os.MkdirTemp(dataDir, ".clone-")
	if err != nil {
		return nil, err
	}
	if err := git.Clone(ctx, f.url, tmp, auth); err != nil {
		_ = os.RemoveAll(tmp)
		return nil, err
	}
	ws, err := repository.AdoptClone(dataDir, tmp, repository.CloneName(f.url), existing)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return nil, err
	}
	if f.token != "" {
		if err := gitx.SaveAuth(f.url, auth); err != nil {
			return ws, fmt.Errorf("cloned, but couldn't save the token: %w", err)
		}
	}
	return ws, nil
}
