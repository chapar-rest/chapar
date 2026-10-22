package gitx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Branch is a local branch, or a remote one with no local branch of the
// same name.
type Branch struct {
	// Name is "main" for a local branch, "origin/main" for a remote one.
	Name    string
	Remote  bool
	Current bool
	// Upstream is the remote branch a local branch tracks, or "".
	Upstream string
}

// Branches lists the local branches, then the remote branches no local
// branch tracks.
func (r *Repo) Branches(ctx context.Context) ([]Branch, error) {
	out, err := r.run(ctx, "for-each-ref", "--format=%(refname)%00%(upstream:short)%00%(HEAD)",
		"refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	var local, remote []Branch
	tracked := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\x00")
		if len(f) < 3 {
			continue
		}
		switch {
		case strings.HasPrefix(f[0], "refs/heads/"):
			b := Branch{Name: strings.TrimPrefix(f[0], "refs/heads/"), Upstream: f[1], Current: f[2] == "*"}
			local = append(local, b)
			tracked[b.Upstream] = true
			tracked["?/"+b.Name] = true
		case strings.HasPrefix(f[0], "refs/remotes/") && !strings.HasSuffix(f[0], "/HEAD"):
			remote = append(remote, Branch{Name: strings.TrimPrefix(f[0], "refs/remotes/"), Remote: true})
		}
	}
	for _, b := range remote {
		_, short, _ := strings.Cut(b.Name, "/")
		if !tracked[b.Name] && !tracked["?/"+short] {
			local = append(local, b)
		}
	}
	return local, nil
}

// Switch checks out a branch. A remote branch, such as "origin/topic",
// gets a local branch that tracks it. Work tree changes stay; git refuses
// when the branch has different versions of the files they touch.
func (r *Repo) Switch(ctx context.Context, b Branch) error {
	if b.Remote {
		_, err := r.run(ctx, "checkout", "--quiet", "--track", b.Name)
		return err
	}
	_, err := r.run(ctx, "checkout", "--quiet", b.Name)
	return err
}

// CreateBranch starts a branch named name at HEAD and checks it out.
func (r *Repo) CreateBranch(ctx context.Context, name string) error {
	if _, err := r.run(ctx, "check-ref-format", "--branch", name); err != nil {
		return fmt.Errorf("%q can't be a branch name", name)
	}
	_, err := r.run(ctx, "checkout", "--quiet", "-b", name)
	return err
}

// Op is a multi-step operation git is in the middle of.
type Op int

const (
	OpNone Op = iota
	OpMerge
	OpRevert
	OpRebase
	OpCherryPick
)

func (o Op) String() string {
	switch o {
	case OpMerge:
		return "merge"
	case OpRevert:
		return "revert"
	case OpRebase:
		return "rebase"
	case OpCherryPick:
		return "cherry-pick"
	}
	return ""
}

// InProgress reports the operation git stopped in, for the user to
// finish or abort, and its message (the merge or revert commit's first
// line).
func (r *Repo) InProgress(ctx context.Context) (Op, string) {
	out, err := r.run(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return OpNone, ""
	}
	dir := strings.TrimSpace(string(out))
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	var op Op
	switch {
	case exists("rebase-merge"), exists("rebase-apply"):
		return OpRebase, ""
	case exists("MERGE_HEAD"):
		op = OpMerge
	case exists("REVERT_HEAD"):
		op = OpRevert
	case exists("CHERRY_PICK_HEAD"):
		return OpCherryPick, ""
	default:
		return OpNone, ""
	}
	msg, _ := os.ReadFile(filepath.Join(dir, "MERGE_MSG"))
	first, _, _ := strings.Cut(string(msg), "\n")
	return op, strings.TrimSpace(first)
}

// Merge merges ref, such as "origin/main", into the current branch. It
// reports whether git stopped with conflicts for the user to resolve; the
// merge is then in progress.
func (r *Repo) Merge(ctx context.Context, ref string) (bool, error) {
	_, err := r.run(ctx, "merge", "--quiet", "--no-edit", ref)
	return r.stopped(ctx, err)
}

// Revert makes a commit that undoes c. A merge commit is undone against
// its first parent. It reports whether git stopped with conflicts.
func (r *Repo) Revert(ctx context.Context, c Commit) (bool, error) {
	args := []string{"revert", "--no-edit"}
	if len(c.Parents) > 1 {
		args = append(args, "-m", "1")
	}
	_, err := r.run(ctx, append(args, c.Hash)...)
	return r.stopped(ctx, err)
}

// stopped tells conflicts, which leave an operation for the user to
// finish, from other failures.
func (r *Repo) stopped(ctx context.Context, err error) (bool, error) {
	if err == nil {
		return false, nil
	}
	if op, _ := r.InProgress(ctx); op != OpNone {
		if st, serr := r.Status(ctx); serr == nil {
			var conflicted []string
			for _, c := range st.Changes {
				if c.Kind == Conflicted {
					conflicted = append(conflicted, c.Path)
				}
			}
			if len(conflicted) > 0 {
				r.keepOurs(ctx, conflicted)
				return true, nil
			}
		}
	}
	return false, err
}

// keepOurs puts this branch's version of each conflicted workspace file
// in the work tree in place of the one git wrote with conflict markers,
// which Chapar can't load. The conflict stays recorded in the index, so
// the file still has to be resolved.
func (r *Repo) keepOurs(ctx context.Context, paths []string) {
	for _, p := range paths {
		if !strings.HasSuffix(p, ".yaml") {
			continue
		}
		// Fails, harmlessly, when this branch deleted the file.
		_, _ = r.run(ctx, "checkout", "--ours", "--", r.spec(p))
	}
}

// Abort gives up the operation in progress and puts the branch and files
// back as they were before it.
func (r *Repo) Abort(ctx context.Context, op Op) error {
	switch op {
	case OpMerge:
		_, err := r.run(ctx, "merge", "--abort")
		return err
	case OpRevert:
		_, err := r.run(ctx, "revert", "--abort")
		return err
	}
	return fmt.Errorf("aborting a %s needs a terminal", op)
}

// Continue records the operation in progress once its conflicts are
// resolved, with git's own commit message for it.
func (r *Repo) Continue(ctx context.Context, op Op) error {
	switch op {
	case OpMerge:
		_, err := r.run(ctx, "commit", "--quiet", "--no-edit")
		return err
	case OpRevert:
		_, err := r.run(ctx, "-c", "core.editor=true", "revert", "--continue")
		return err
	}
	return fmt.Errorf("finishing a %s needs a terminal", op)
}

// Versions are the three versions of a conflicted file. A version is nil
// when that side doesn't have the file.
type Versions struct {
	Base, Ours, Theirs []byte
}

// ConflictVersions reads the common ancestor's, this branch's and the
// incoming version of a conflicted file.
func (r *Repo) ConflictVersions(ctx context.Context, path string) (Versions, error) {
	var v Versions
	for stage, dst := range map[int]*[]byte{1: &v.Base, 2: &v.Ours, 3: &v.Theirs} {
		out, err := r.run(ctx, "show", fmt.Sprintf(":%d:%s", stage, r.spec(path)))
		if err != nil {
			var ex *ExitError
			if errors.As(err, &ex) {
				continue // that side has no such file
			}
			return v, err
		}
		*dst = out
	}
	if v.Ours == nil && v.Theirs == nil {
		return v, fmt.Errorf("%s is not in conflict", path)
	}
	return v, nil
}

// Resolve settles a conflicted file with data, or deletes it when data
// is nil, and marks it resolved.
func (r *Repo) Resolve(ctx context.Context, path string, data []byte) error {
	file := filepath.Join(r.Dir, filepath.FromSlash(path))
	if data == nil {
		_, err := r.run(ctx, "rm", "--quiet", "--force", "--", r.spec(path))
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return err
	}
	_, err := r.run(ctx, "add", "--", r.spec(path))
	return err
}
