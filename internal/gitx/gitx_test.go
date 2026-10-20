package gitx

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo makes a git repository in a temp dir with the user's git config
// shut out, and returns the dir and a function that runs git in it.
func newRepo(t *testing.T) (string, func(args ...string)) {
	t.Helper()
	g, err := Find(context.Background())
	if err != nil {
		t.Skip("git is not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Ada")
	t.Setenv("GIT_AUTHOR_EMAIL", "ada@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Ada")
	t.Setenv("GIT_COMMITTER_EMAIL", "ada@example.com")
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command(g.path, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q", "-b", "main")
	return dir, git
}

func write(t *testing.T, dir, path, data string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func open(t *testing.T, dir string) *Repo {
	t.Helper()
	g, err := Find(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	r, err := g.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func byPath(changes []Change) map[string]Change {
	m := map[string]Change{}
	for _, c := range changes {
		m[c.Path] = c
	}
	return m
}

func TestOpenOutsideRepo(t *testing.T) {
	g, err := Find(context.Background())
	if err != nil {
		t.Skip("git is not installed")
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	_, err = g.Open(context.Background(), t.TempDir())
	if !errors.Is(err, ErrNotRepo) {
		t.Fatalf("Open = %v, want ErrNotRepo", err)
	}
}

func TestStatusBeforeFirstCommit(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "envs/Stage.yaml", "kind: Environment\n")
	write(t, dir, "requests/a b.yaml", "kind: Request\n")
	git("add", "requests/a b.yaml")

	ctx := context.Background()
	r := open(t, dir)
	st, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.Branch != "main" || st.Head != "" {
		t.Errorf("branch %q head %q, want main and no head", st.Branch, st.Head)
	}
	got := byPath(st.Changes)
	if got["envs/Stage.yaml"].Kind != Untracked || got["requests/a b.yaml"].Kind != Added {
		t.Fatalf("changes = %+v", st.Changes)
	}
	if commits, err := r.Log(ctx, 10); err != nil || len(commits) != 0 {
		t.Errorf("Log = %v, %v; want none", commits, err)
	}
	// Before the first commit an added file diffs against the empty tree.
	d, err := r.Diff(ctx, got["requests/a b.yaml"])
	if err != nil || !strings.Contains(string(d), "+kind: Request") {
		t.Errorf("Diff = %q, %v", d, err)
	}
}

func TestStatusKinds(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "keep.yaml", "a: 1\n")
	write(t, dir, "edit.yaml", "a: 1\n")
	write(t, dir, "gone.yaml", "a: 1\n")
	write(t, dir, "old name.yaml", "a: 1\nb: 2\nc: 3\n")
	git("add", ".")
	git("commit", "-q", "-m", "first")

	write(t, dir, "edit.yaml", "a: 2\n")
	if err := os.Remove(filepath.Join(dir, "gone.yaml")); err != nil {
		t.Fatal(err)
	}
	git("mv", "old name.yaml", "new name.yaml")
	write(t, dir, "new.yaml", "x: 1\ny: 2")

	ctx := context.Background()
	r := open(t, dir)
	st, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.Head == "" {
		t.Error("head is empty after a commit")
	}
	got := byPath(st.Changes)
	if len(got) != 4 {
		t.Fatalf("changes = %+v", st.Changes)
	}
	want := map[string]ChangeKind{"edit.yaml": Modified, "gone.yaml": Deleted, "new name.yaml": Renamed, "new.yaml": Untracked}
	for p, k := range want {
		if got[p].Kind != k {
			t.Errorf("%s: kind %v, want %v", p, got[p].Kind, k)
		}
	}
	if o := got["new name.yaml"].OrigPath; o != "old name.yaml" {
		t.Errorf("rename from %q", o)
	}

	d, err := r.Diff(ctx, got["edit.yaml"])
	if err != nil || !strings.Contains(string(d), "-a: 1\n+a: 2") {
		t.Errorf("Diff edit = %q, %v", d, err)
	}
	d, err = r.Diff(ctx, got["new.yaml"])
	if err != nil || !strings.Contains(string(d), "@@ -0,0 +1,2 @@\n+x: 1\n+y: 2\n\\ No newline") {
		t.Errorf("Diff untracked = %q, %v", d, err)
	}
	d, err = r.Diff(ctx, got["new name.yaml"])
	if err != nil || !strings.Contains(string(d), "rename from old name.yaml") {
		t.Errorf("Diff rename = %q, %v", d, err)
	}
}

// A workspace below the top of the work tree sees only its own files, with
// paths relative to itself.
func TestWorkspaceInsideBiggerRepo(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "Tutorial/envs/Stage.yaml", "a: 1\n")
	write(t, dir, "Other/envs/Prod.yaml", "a: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "first")
	write(t, dir, "Tutorial/envs/Stage.yaml", "a: 2\n")
	write(t, dir, "Other/envs/Prod.yaml", "a: 2\n")
	git("commit", "-q", "-am", "second")
	write(t, dir, "Tutorial/envs/Stage.yaml", "a: 3\n")
	write(t, dir, "Other/envs/Prod.yaml", "a: 3\n")

	ctx := context.Background()
	r := open(t, filepath.Join(dir, "Tutorial"))
	if r.Contained() {
		t.Error("Contained = true for a workspace below the root")
	}
	st, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Changes) != 1 || st.Changes[0].Path != "envs/Stage.yaml" {
		t.Fatalf("changes = %+v", st.Changes)
	}

	commits, err := r.Log(ctx, 10)
	if err != nil || len(commits) != 2 {
		t.Fatalf("Log = %+v, %v", commits, err)
	}
	if commits[0].Subject != "second" || commits[0].Author != "Ada" || len(commits[0].Parents) != 1 {
		t.Errorf("newest commit = %+v", commits[0])
	}
	for _, c := range commits {
		changes, err := r.CommitChanges(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		if len(changes) != 1 || changes[0].Path != "envs/Stage.yaml" {
			t.Errorf("%s changes = %+v", c.Subject, changes)
		}
		d, err := r.CommitDiff(ctx, c, changes[0])
		if err != nil || !strings.Contains(string(d), "Tutorial/envs/Stage.yaml") || strings.Contains(string(d), "Prod") {
			t.Errorf("%s diff = %q, %v", c.Subject, d, err)
		}
	}
	if root := commits[1]; len(root.Parents) != 0 {
		t.Errorf("root commit parents = %v", root.Parents)
	}

	old, err := r.ShowFile(ctx, "HEAD", "envs/Stage.yaml")
	if err != nil || string(old) != "a: 2\n" {
		t.Errorf("ShowFile = %q, %v", old, err)
	}
}

func TestStatusAheadBehind(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "a.yaml", "a: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "first")
	git("branch", "base")
	git("branch", "--set-upstream-to=base")
	write(t, dir, "a.yaml", "a: 2\n")
	git("commit", "-q", "-am", "second")

	st, err := open(t, dir).Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Upstream != "base" || st.Ahead != 1 || st.Behind != 0 {
		t.Errorf("upstream %q ahead %d behind %d", st.Upstream, st.Ahead, st.Behind)
	}
}

func TestParseStatusConflict(t *testing.T) {
	r := &Repo{}
	out := "# branch.oid abc\x00# branch.head (detached)\x00" +
		"u UU N... 100644 100644 100644 100644 a b c envs/Stage.yaml\x00"
	st := r.parseStatus([]byte(out))
	if st.Branch != "" || st.Head != "abc" {
		t.Errorf("branch %q head %q", st.Branch, st.Head)
	}
	if len(st.Changes) != 1 || st.Changes[0].Kind != Conflicted || st.Changes[0].Path != "envs/Stage.yaml" {
		t.Errorf("changes = %+v", st.Changes)
	}
}
