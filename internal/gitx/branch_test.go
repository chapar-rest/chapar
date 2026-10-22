package gitx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchesAndSwitch(t *testing.T) {
	bare, dirA, gitA := remotePair(t)
	ctx := context.Background()
	gitA("checkout", "-q", "-b", "topic")
	write(t, dirA, "t.yaml", "t: 1\n")
	gitA("add", ".")
	gitA("commit", "-q", "-m", "topic work")
	gitA("push", "-q", "origin", "topic")

	g, _ := Find(ctx)
	dirB := filepath.Join(t.TempDir(), "b")
	if err := g.Clone(ctx, bare, dirB, Auth{}); err != nil {
		t.Fatal(err)
	}
	b := open(t, dirB)
	branches, err := b.Branches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, br := range branches {
		names = append(names, br.Name)
	}
	if got := strings.Join(names, ","); got != "main,origin/topic" {
		t.Fatalf("branches = %s", got)
	}
	if !branches[0].Current || branches[0].Upstream != "origin/main" || !branches[1].Remote {
		t.Errorf("branches = %+v", branches)
	}

	if err := b.Switch(ctx, branches[1]); err != nil {
		t.Fatal(err)
	}
	if st := status(t, b); st.Branch != "topic" || st.Upstream != "origin/topic" {
		t.Errorf("after switch: %q tracking %q", st.Branch, st.Upstream)
	}
	if _, err := os.Stat(filepath.Join(dirB, "t.yaml")); err != nil {
		t.Error("topic's file is missing after the switch")
	}

	if err := b.CreateBranch(ctx, "bad name"); err == nil {
		t.Error("created a branch with a space in its name")
	}
	if err := b.CreateBranch(ctx, "fix/headers"); err != nil {
		t.Fatal(err)
	}
	if st := status(t, b); st.Branch != "fix/headers" {
		t.Errorf("branch = %q", st.Branch)
	}
}

func TestMergeConflictResolve(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "env.yaml", "a: 1\nb: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("checkout", "-q", "-b", "other")
	write(t, dir, "env.yaml", "a: 1\nb: theirs\n")
	git("commit", "-q", "-am", "theirs")
	git("checkout", "-q", "main")
	write(t, dir, "env.yaml", "a: 1\nb: ours\n")
	git("commit", "-q", "-am", "ours")

	ctx := context.Background()
	r := open(t, dir)
	conflicts, err := r.Merge(ctx, "other")
	if err != nil || !conflicts {
		t.Fatalf("Merge = %v, %v; want conflicts", conflicts, err)
	}
	if op, msg := r.InProgress(ctx); op != OpMerge || msg != "Merge branch 'other'" {
		t.Errorf("in progress = %v %q", op, msg)
	}
	got := byPath(status(t, r).Changes)
	if got["env.yaml"].Kind != Conflicted {
		t.Fatalf("changes = %+v", got)
	}
	// The work tree keeps a loadable file: this branch's version, with no
	// conflict markers.
	if data, _ := os.ReadFile(filepath.Join(dir, "env.yaml")); string(data) != "a: 1\nb: ours\n" {
		t.Errorf("work tree env.yaml = %q", data)
	}
	v, err := r.ConflictVersions(ctx, "env.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if string(v.Base) != "a: 1\nb: 1\n" || string(v.Ours) != "a: 1\nb: ours\n" || string(v.Theirs) != "a: 1\nb: theirs\n" {
		t.Errorf("versions = %q %q %q", v.Base, v.Ours, v.Theirs)
	}
	if err := r.Continue(ctx, OpMerge); err == nil {
		t.Error("finished a merge with a conflict left")
	}
	if err := r.Resolve(ctx, "env.yaml", []byte("a: 1\nb: both\n")); err != nil {
		t.Fatal(err)
	}
	if err := r.Continue(ctx, OpMerge); err != nil {
		t.Fatal(err)
	}
	if op, _ := r.InProgress(ctx); op != OpNone {
		t.Errorf("still in %v", op)
	}
	if parents := gitOut(t, dir, "log", "-1", "--format=%P"); len(strings.Fields(parents)) != 2 {
		t.Errorf("merge commit parents = %q", parents)
	}
	// The history lists the merge and both branches' commits.
	commits, _ := r.Log(ctx, 10)
	var subjects []string
	for _, c := range commits {
		subjects = append(subjects, c.Subject)
	}
	if got := strings.Join(subjects, ","); got != "Merge branch 'other',ours,theirs,base" {
		t.Errorf("log = %s", got)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "env.yaml")); string(data) != "a: 1\nb: both\n" {
		t.Errorf("env.yaml = %q", data)
	}
}

func TestMergeAbort(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "x.yaml", "v: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("checkout", "-q", "-b", "other")
	write(t, dir, "x.yaml", "v: 2\n")
	git("commit", "-q", "-am", "2")
	git("checkout", "-q", "main")
	write(t, dir, "x.yaml", "v: 3\n")
	git("commit", "-q", "-am", "3")

	ctx := context.Background()
	r := open(t, dir)
	if c, err := r.Merge(ctx, "other"); err != nil || !c {
		t.Fatalf("Merge = %v, %v", c, err)
	}
	if err := r.Abort(ctx, OpMerge); err != nil {
		t.Fatal(err)
	}
	if op, _ := r.InProgress(ctx); op != OpNone {
		t.Errorf("still in %v", op)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "x.yaml")); string(data) != "v: 3\n" {
		t.Errorf("x.yaml = %q", data)
	}
	// A clean merge needs no resolving.
	git("checkout", "-q", "-b", "side", "HEAD~1")
	write(t, dir, "y.yaml", "y: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "y")
	git("checkout", "-q", "main")
	if c, err := r.Merge(ctx, "side"); err != nil || c {
		t.Errorf("clean Merge = %v, %v", c, err)
	}
	// A ref that doesn't exist is an error, not a conflict.
	if _, err := r.Merge(ctx, "nope"); err == nil {
		t.Error("merged a missing branch")
	}
}

func TestRevert(t *testing.T) {
	dir, git := newRepo(t)
	write(t, dir, "x.yaml", "v: 1\n")
	git("add", ".")
	git("commit", "-q", "-m", "one")
	write(t, dir, "x.yaml", "v: 2\n")
	git("commit", "-q", "-am", "two")
	write(t, dir, "x.yaml", "v: 3\n")
	git("commit", "-q", "-am", "three")

	ctx := context.Background()
	r := open(t, dir)
	commits, _ := r.Log(ctx, 10)
	// Reverting "three" applies cleanly.
	if c, err := r.Revert(ctx, commits[0]); err != nil || c {
		t.Fatalf("Revert three = %v, %v", c, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "x.yaml")); string(data) != "v: 2\n" {
		t.Errorf("x.yaml = %q", data)
	}
	// Reverting "two" now conflicts with the revert of three.
	write(t, dir, "x.yaml", "v: 9\n")
	git("commit", "-q", "-am", "nine")
	if c, err := r.Revert(ctx, commits[1]); err != nil || !c {
		t.Fatalf("Revert two = %v, %v", c, err)
	}
	if op, _ := r.InProgress(ctx); op != OpRevert {
		t.Fatalf("in progress = %v", op)
	}
	if err := r.Resolve(ctx, "x.yaml", []byte("v: 1\n")); err != nil {
		t.Fatal(err)
	}
	if err := r.Continue(ctx, OpRevert); err != nil {
		t.Fatal(err)
	}
	if msg := gitOut(t, dir, "log", "-1", "--format=%s"); !strings.HasPrefix(msg, `Revert "two"`) {
		t.Errorf("revert commit = %q", msg)
	}
}
