package gitx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Init makes dir the root of a new repository. A git that has no default
// branch configured starts on main.
func (g *Git) Init(ctx context.Context, dir string) error {
	args := []string{"init", "--quiet"}
	if out, _ := g.command(ctx, dir, "config", "--get", "init.defaultBranch").Output(); strings.TrimSpace(string(out)) == "" {
		args = append([]string{"-c", "init.defaultBranch=main"}, args...)
	}
	cmd := g.command(ctx, dir, args...)
	return runCmd(cmd, args)
}

// Remote is the repository's remote: origin, or the first one when there
// is no origin. Both are "" when the repository has none.
func (r *Repo) Remote(ctx context.Context) (name, url string, err error) {
	out, err := r.run(ctx, "remote")
	if err != nil {
		return "", "", err
	}
	names := strings.Fields(string(out))
	if len(names) == 0 {
		return "", "", nil
	}
	name = names[0]
	for _, n := range names {
		if n == "origin" {
			name = n
		}
	}
	out, err = r.run(ctx, "remote", "get-url", name)
	if err != nil {
		return "", "", err
	}
	return name, strings.TrimSpace(string(out)), nil
}

// SetRemote points origin, or the remote Remote returns, at url, adding
// origin when the repository has no remote.
func (r *Repo) SetRemote(ctx context.Context, url string) error {
	name, _, err := r.Remote(ctx)
	if err != nil {
		return err
	}
	if name == "" {
		_, err = r.run(ctx, "remote", "add", "origin", url)
		return err
	}
	_, err = r.run(ctx, "remote", "set-url", name, url)
	return err
}

// Identity is the name and email commits are made with, from any level of
// git config. Either is "" when it is not set.
func (r *Repo) Identity(ctx context.Context) (name, email string) {
	n, _ := r.run(ctx, "config", "--get", "user.name")
	e, _ := r.run(ctx, "config", "--get", "user.email")
	return strings.TrimSpace(string(n)), strings.TrimSpace(string(e))
}

// SetIdentity sets the name and email for commits in this repository only.
func (r *Repo) SetIdentity(ctx context.Context, name, email string) error {
	if _, err := r.run(ctx, "config", "user.name", name); err != nil {
		return err
	}
	_, err := r.run(ctx, "config", "user.email", email)
	return err
}

// paths lists the work tree paths a change touches: both sides of a
// rename.
func (r *Repo) paths(changes []Change) []string {
	var out []string
	for _, c := range changes {
		out = append(out, r.spec(c.Path))
		if c.OrigPath != "" {
			out = append(out, r.spec(c.OrigPath))
		}
	}
	return out
}

// Commit records the changes, and only them, with message. Other changes,
// staged or not, are left as they are. It returns the new commit's id.
func (r *Repo) Commit(ctx context.Context, message string, changes []Change) (string, error) {
	if len(changes) == 0 {
		return "", errors.New("nothing to commit")
	}
	for _, c := range changes {
		if c.Kind == Conflicted {
			return "", fmt.Errorf("%s has a merge conflict; resolve it before committing", c.Path)
		}
	}
	paths := r.paths(changes)
	// Stage first: commit with paths only takes files git already knows.
	// A file gone from the work tree is staged as removed; git add would
	// reject the old name of a rename that is already staged.
	var present, gone []string
	for _, p := range paths {
		if _, err := os.Lstat(filepath.Join(r.Root, filepath.FromSlash(p))); err == nil {
			present = append(present, p)
		} else {
			gone = append(gone, p)
		}
	}
	if len(present) > 0 {
		if _, err := r.run(ctx, append([]string{"add", "--"}, present...)...); err != nil {
			return "", err
		}
	}
	if len(gone) > 0 {
		if _, err := r.run(ctx, append([]string{"rm", "--quiet", "--cached", "--ignore-unmatch", "--"}, gone...)...); err != nil {
			return "", err
		}
	}
	args := append([]string{"commit", "--quiet", "--file=-", "--only", "--"}, paths...)
	cmd := r.git.command(ctx, r.Root, args...)
	cmd.Stdin = strings.NewReader(message)
	if err := runCmd(cmd, args); err != nil {
		return "", err
	}
	out, err := r.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Discard puts a changed file back the way the last commit has it: an
// edited or deleted file is restored, a new file is deleted.
func (r *Repo) Discard(ctx context.Context, c Change) error {
	switch c.Kind {
	case Conflicted:
		return fmt.Errorf("%s has a merge conflict; resolve it before discarding", c.Path)
	case Untracked:
		return os.Remove(filepath.Join(r.Dir, filepath.FromSlash(c.Path)))
	case Added:
		_, err := r.run(ctx, "rm", "--quiet", "--force", "--", r.spec(c.Path))
		return err
	case Renamed:
		if _, err := r.run(ctx, "rm", "--quiet", "--force", "--", r.spec(c.Path)); err != nil {
			return err
		}
		if c.OrigPath == "" {
			return nil
		}
		_, err := r.run(ctx, "checkout", "HEAD", "--", r.spec(c.OrigPath))
		return err
	}
	_, err := r.run(ctx, "checkout", "HEAD", "--", r.spec(c.Path))
	return err
}
