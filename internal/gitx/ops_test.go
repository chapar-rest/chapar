package gitx

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Remote commands name the running binary as git's askpass helper; in
// tests that is the test binary, so answer like chapar would.
func TestMain(m *testing.M) {
	if os.Getenv(AskPassEnv) == "1" {
		os.Exit(AskPass(os.Args[1:], os.Stdout))
	}
	os.Exit(m.Run())
}

func status(t *testing.T, r *Repo) *Status {
	t.Helper()
	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestInitStartsOnMain(t *testing.T) {
	newRepo(t) // sets up an isolated git config
	g, _ := Find(context.Background())
	dir := t.TempDir()
	if err := g.Init(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	if st := status(t, open(t, dir)); st.Branch != "main" {
		t.Errorf("branch = %q, want main", st.Branch)
	}
}

func TestCommitOnlySelected(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "a.yaml", "a: 1\n")
	write(t, dir, "b.yaml", "b: 1\n")
	write(t, dir, "gone.yaml", "g: 1\n")
	write(t, dir, "old.yaml", "o: 1\np: 2\nq: 3\n")
	git("add", ".")
	git("commit", "-q", "-m", "first")

	write(t, dir, "a.yaml", "a: 2\n")
	write(t, dir, "b.yaml", "b: 2\n")
	git("add", "b.yaml") // staged, but not picked: must stay staged and uncommitted
	if err := os.Remove(filepath.Join(dir, "gone.yaml")); err != nil {
		t.Fatal(err)
	}
	git("mv", "old.yaml", "new.yaml")
	write(t, dir, "fresh.yaml", "f: 1\n")

	ctx := context.Background()
	r := open(t, dir)
	got := byPath(status(t, r).Changes)
	pick := []Change{got["a.yaml"], got["gone.yaml"], got["new.yaml"], got["fresh.yaml"]}
	hash, err := r.Commit(ctx, "Pick some\n\nwith a body", pick)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("no commit id")
	}
	left := status(t, r).Changes
	if len(left) != 1 || left[0].Path != "b.yaml" || left[0].Staged != 'M' {
		t.Fatalf("left = %+v, want b.yaml still staged", left)
	}
	if msg := gitOut(t, dir, "log", "-1", "--format=%B"); msg != "Pick some\n\nwith a body" {
		t.Errorf("message = %q", msg)
	}
	files := gitOut(t, dir, "show", "--name-status", "--format=", "HEAD")
	for _, want := range []string{"M\ta.yaml", "D\tgone.yaml", "A\tfresh.yaml", "R100\told.yaml\tnew.yaml"} {
		if !strings.Contains(files, want) {
			t.Errorf("commit files %q lack %q", files, want)
		}
	}
}

func TestFirstCommit(t *testing.T) {
	dir, _ := newRepo(t)
	write(t, dir, "a.yaml", "a: 1\n")
	write(t, dir, "b.yaml", "b: 1\n")
	r := open(t, dir)
	got := byPath(status(t, r).Changes)
	if _, err := r.Commit(context.Background(), "first", []Change{got["a.yaml"]}); err != nil {
		t.Fatal(err)
	}
	left := status(t, r).Changes
	if len(left) != 1 || left[0].Path != "b.yaml" {
		t.Errorf("left = %+v", left)
	}
}

func TestDiscard(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "edit.yaml", "e: 1\n")
	write(t, dir, "gone.yaml", "g: 1\n")
	write(t, dir, "old.yaml", "o: 1\np: 2\nq: 3\n")
	git("add", ".")
	git("commit", "-q", "-m", "first")

	write(t, dir, "edit.yaml", "e: 2\n")
	git("add", "edit.yaml")
	write(t, dir, "edit.yaml", "e: 3\n")
	git("rm", "-q", "gone.yaml")
	git("mv", "old.yaml", "new.yaml")
	write(t, dir, "fresh.yaml", "f: 1\n")
	write(t, dir, "added.yaml", "a: 1\n")
	git("add", "added.yaml")

	ctx := context.Background()
	r := open(t, dir)
	for _, c := range status(t, r).Changes {
		if err := r.Discard(ctx, c); err != nil {
			t.Fatalf("discard %s: %v", c.Path, err)
		}
	}
	if left := status(t, r).Changes; len(left) != 0 {
		t.Fatalf("left = %+v", left)
	}
	for _, p := range []string{"fresh.yaml", "added.yaml", "new.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, p)); !os.IsNotExist(err) {
			t.Errorf("%s still exists", p)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "edit.yaml")); string(b) != "e: 1\n" {
		t.Errorf("edit.yaml = %q", b)
	}
}

// remotePair makes a bare repository and a clone of it with one commit
// pushed, and returns the bare repository's path and the clone.
func remotePair(t *testing.T) (string, string, func(args ...string)) {
	t.Helper()
	dir, git := newRepo(t)
	bare := t.TempDir()
	gitOut(t, bare, "init", "-q", "--bare", "-b", "main")
	write(t, dir, "a.yaml", "a: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "first")

	ctx := context.Background()
	r := open(t, dir)
	if name, _, _ := r.Remote(ctx); name != "" {
		t.Fatalf("remote %q before SetRemote", name)
	}
	if err := r.SetRemote(ctx, bare); err != nil {
		t.Fatal(err)
	}
	if name, url, _ := r.Remote(ctx); name != "origin" || url != bare {
		t.Fatalf("remote = %q %q", name, url)
	}
	st := status(t, r)
	if err := r.Push(ctx, Auth{}, st); err != nil {
		t.Fatal(err)
	}
	if st := status(t, r); st.Upstream != "origin/main" {
		t.Fatalf("upstream after first push = %q", st.Upstream)
	}
	return bare, dir, git
}

func TestPushPull(t *testing.T) {
	bare, dirA, gitA := remotePair(t)
	ctx := context.Background()
	g, _ := Find(ctx)

	dirB := filepath.Join(t.TempDir(), "b")
	if err := g.Clone(ctx, bare, dirB, Auth{}); err != nil {
		t.Fatal(err)
	}
	b := open(t, dirB)

	write(t, dirA, "a.yaml", "a: 2\n")
	gitA("commit", "-q", "-am", "second")
	a := open(t, dirA)
	if err := a.Push(ctx, Auth{}, status(t, a)); err != nil {
		t.Fatal(err)
	}

	moved, err := b.Pull(ctx, Auth{}, status(t, b))
	if err != nil || !moved {
		t.Fatalf("Pull = %v, %v", moved, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dirB, "a.yaml")); string(data) != "a: 2\n" {
		t.Errorf("after pull a.yaml = %q", data)
	}
	if moved, err := b.Pull(ctx, Auth{}, status(t, b)); err != nil || moved {
		t.Errorf("second Pull = %v, %v", moved, err)
	}

	// Both sides commit: B's push is rejected and its pull can't
	// fast-forward.
	write(t, dirA, "a.yaml", "a: 3\n")
	gitA("commit", "-q", "-am", "third")
	if err := a.Push(ctx, Auth{}, status(t, a)); err != nil {
		t.Fatal(err)
	}
	write(t, dirB, "b.yaml", "b: 1\n")
	gitOut(t, dirB, "add", ".")
	gitOut(t, dirB, "commit", "-q", "-m", "from b")
	if err := b.Push(ctx, Auth{}, status(t, b)); !errors.Is(err, ErrRejected) {
		t.Errorf("Push = %v, want ErrRejected", err)
	}
	if err := b.Fetch(ctx, Auth{}); err != nil {
		t.Fatal(err)
	}
	if st := status(t, b); st.Ahead != 1 || st.Behind != 1 {
		t.Errorf("after fetch ahead %d behind %d", st.Ahead, st.Behind)
	}
	if _, err := b.Pull(ctx, Auth{}, status(t, b)); !errors.Is(err, ErrDiverged) {
		t.Errorf("Pull = %v, want ErrDiverged", err)
	}
}

// A new local repository connected to a remote that already has history
// takes the remote branch on its first pull.
func TestPullIntoEmptyRepo(t *testing.T) {
	bare, _, _ := remotePair(t)
	ctx := context.Background()
	g, _ := Find(ctx)
	dir := t.TempDir()
	if err := g.Init(ctx, dir); err != nil {
		t.Fatal(err)
	}
	r := open(t, dir)
	if err := r.SetRemote(ctx, bare); err != nil {
		t.Fatal(err)
	}
	moved, err := r.Pull(ctx, Auth{}, status(t, r))
	if err != nil || !moved {
		t.Fatalf("Pull = %v, %v", moved, err)
	}
	if st := status(t, r); st.Upstream != "origin/main" || st.Head == "" {
		t.Errorf("after pull: %+v", st)
	}

	// A branch the remote doesn't have can't be pulled.
	gitOut(t, dir, "checkout", "-q", "-b", "topic")
	if _, err := r.Pull(ctx, Auth{}, status(t, r)); !errors.Is(err, ErrNoUpstream) {
		t.Errorf("Pull topic = %v, want ErrNoUpstream", err)
	}
}

func TestIdentity(t *testing.T) {
	dir, _ := newRepo(t)
	t.Setenv("GIT_AUTHOR_NAME", "")
	r := open(t, dir)
	ctx := context.Background()
	if n, e := r.Identity(ctx); n != "" || e != "" {
		t.Errorf("identity before = %q %q", n, e)
	}
	if err := r.SetIdentity(ctx, "Grace", "grace@example.com"); err != nil {
		t.Fatal(err)
	}
	if n, e := r.Identity(ctx); n != "Grace" || e != "grace@example.com" {
		t.Errorf("identity after = %q %q", n, e)
	}
}

func TestAskPass(t *testing.T) {
	t.Setenv(askUserEnv, "ada")
	t.Setenv(askSecretEnv, "tok")
	for prompt, want := range map[string]string{
		"Username for 'https://github.com': ":     "ada\n",
		"Password for 'https://ada@github.com': ": "tok\n",
	} {
		var out bytes.Buffer
		if code := AskPass([]string{prompt}, &out); code != 0 || out.String() != want {
			t.Errorf("%q: code %d out %q", prompt, code, out.String())
		}
	}
	var out bytes.Buffer
	if code := AskPass([]string{"Are you sure you want to continue connecting (yes/no)? "}, &out); code == 0 {
		t.Errorf("unknown prompt answered %q", out.String())
	}
	t.Setenv(askSecretEnv, "")
	if code := AskPass([]string{"Password for 'https://x': "}, &out); code == 0 {
		t.Error("answered with no token")
	}
}

func TestRemoteError(t *testing.T) {
	cases := map[string]error{
		"could not read Username for 'https://github.com': terminal prompts disabled": ErrAuth,
		"Authentication failed for 'https://github.com/a/b.git/'":                     ErrAuth,
		"Not possible to fast-forward, aborting.":                                     ErrDiverged,
		"failed to push some refs\n ! [rejected] main -> main (fetch first)":          ErrRejected,
	}
	for msg, want := range cases {
		if err := remoteError(&ExitError{Args: []string{"x"}, Code: 1, Message: msg}); !errors.Is(err, want) {
			t.Errorf("%q -> %v, want %v", msg, err, want)
		}
	}
}

func TestCredKey(t *testing.T) {
	for remote, want := range map[string]string{
		"https://GitHub.com/a/b.git":     "https://github.com",
		"https://ada@github.com/a/b.git": "https://github.com",
		"http://localhost:3000/x":        "http://localhost:3000",
		"git@github.com:a/b.git":         "",
		"ssh://git@github.com/a/b.git":   "",
		"/srv/repo.git":                  "",
	} {
		if got, _ := credKey(remote); got != want {
			t.Errorf("credKey(%q) = %q, want %q", remote, got, want)
		}
	}
}
