package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// HasCommits reports whether HEAD points at a commit. It is false in a
// repository nobody has committed to yet.
func (r *Repo) HasCommits(ctx context.Context) bool {
	_, err := r.run(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// Diff is the patch from the last commit to the work tree for one change,
// staged and unstaged edits together.
func (r *Repo) Diff(ctx context.Context, c Change) ([]byte, error) {
	if c.Kind == Untracked {
		return r.newFileDiff(c.Path)
	}
	base := "HEAD"
	if !r.HasCommits(ctx) {
		tree, err := r.emptyTree(ctx)
		if err != nil {
			return nil, err
		}
		base = tree
	}
	args := []string{"diff", "--no-color", "--no-ext-diff", "-M", base, "--", r.spec(c.Path)}
	if c.OrigPath != "" {
		args = append(args, r.spec(c.OrigPath))
	}
	return r.run(ctx, args...)
}

// emptyTree is the id of the empty tree, the base to diff against before
// the first commit. It depends on the repository's hash function.
func (r *Repo) emptyTree(ctx context.Context) (string, error) {
	cmd := r.git.command(ctx, r.Root, "hash-object", "-t", "tree", "--stdin")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.Output()
	if err != nil {
		return "", runError([]string{"hash-object"}, err, nil)
	}
	return strings.TrimSpace(string(out)), nil
}

// newFileDiff writes an untracked file as a patch that adds it. git can
// do this with --no-index against /dev/null, but that path does not exist
// on every platform.
func (r *Repo) newFileDiff(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(path)))
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "diff --git a/%s b/%s\nnew file\n", path, path)
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		fmt.Fprintf(&b, "Binary file b/%s\n", path)
		return b.Bytes(), nil
	}
	if len(data) == 0 {
		return b.Bytes(), nil
	}
	text := strings.TrimSuffix(string(data), "\n")
	lines := strings.Split(text, "\n")
	fmt.Fprintf(&b, "--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n", path, len(lines))
	for _, l := range lines {
		b.WriteByte('+')
		b.WriteString(l)
		b.WriteByte('\n')
	}
	if !strings.HasSuffix(string(data), "\n") {
		b.WriteString("\\ No newline at end of file\n")
	}
	return b.Bytes(), nil
}

// ShowFile reads a file as it was at rev, such as "HEAD".
func (r *Repo) ShowFile(ctx context.Context, rev, path string) ([]byte, error) {
	return r.run(ctx, "show", rev+":"+r.spec(path))
}

// Commit is one entry of the history.
type Commit struct {
	Hash    string
	Short   string
	Parents []string
	Author  string
	Email   string
	When    time.Time
	Subject string
}

// Log returns up to n commits that touched the workspace, newest first.
// It returns no commits, and no error, before the first commit.
func (r *Repo) Log(ctx context.Context, n int) ([]Commit, error) {
	if !r.HasCommits(ctx) {
		return nil, nil
	}
	args := []string{"log", "-z", "--no-color", fmt.Sprintf("-n%d", n),
		"--format=%H%x1f%h%x1f%P%x1f%an%x1f%ae%x1f%at%x1f%s"}
	if !r.Contained() {
		// Limiting the log to a path simplifies history: a merge whose
		// result matches one parent would be left out, and the commits it
		// merged with it.
		args = append(args, "--full-history", "--", r.spec(""))
	}
	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseLog(out), nil
}

func parseLog(out []byte) []Commit {
	var commits []Commit
	for _, rec := range bytes.Split(out, []byte{0}) {
		f := strings.Split(strings.TrimPrefix(string(rec), "\n"), "\x1f")
		if len(f) < 7 {
			continue
		}
		var when time.Time
		if sec, err := strconv.ParseInt(f[5], 10, 64); err == nil {
			when = time.Unix(sec, 0)
		}
		commits = append(commits, Commit{
			Hash:    f[0],
			Short:   f[1],
			Parents: strings.Fields(f[2]),
			Author:  f[3],
			Email:   f[4],
			When:    when,
			Subject: f[6],
		})
	}
	return commits
}

// commitDiff is the command that diffs c against its first parent, or
// against nothing for a root commit.
func commitDiff(c Commit, args ...string) []string {
	if len(c.Parents) == 0 {
		return append(append([]string{"diff-tree", "-r", "--root", "--no-commit-id"}, args...), c.Hash)
	}
	return append(append([]string{"diff"}, args...), c.Parents[0], c.Hash)
}

// CommitChanges lists the workspace files c changed, compared with its
// first parent.
func (r *Repo) CommitChanges(ctx context.Context, c Commit) ([]Change, error) {
	args := commitDiff(c, "-z", "--no-color", "-M", "--name-status")
	out, err := r.run(ctx, append(args, "--", r.spec(""))...)
	if err != nil {
		return nil, err
	}
	return r.parseNameStatus(out), nil
}

// parseNameStatus reads `git diff -z --name-status`: a status field, then
// one path, or two for a rename or copy.
func (r *Repo) parseNameStatus(out []byte) []Change {
	f := bytes.Split(out, []byte{0})
	var changes []Change
	for i := 0; i < len(f); i++ {
		st := string(f[i])
		if st == "" || i+1 >= len(f) {
			continue
		}
		c := Change{Staged: st[0], Unstaged: '.'}
		switch st[0] {
		case 'A':
			c.Kind = Added
		case 'D':
			c.Kind = Deleted
		case 'R', 'C':
			c.Kind = Renamed
			if i+2 >= len(f) {
				i = len(f)
				continue
			}
			i++
			c.OrigPath, _ = r.rel(string(f[i]))
			if st[0] == 'C' {
				c.Kind = Added
			}
		default:
			c.Kind = Modified
		}
		i++
		p, ok := r.rel(string(f[i]))
		if !ok {
			continue
		}
		c.Path = p
		changes = append(changes, c)
	}
	return changes
}

// CommitDiff is the patch c made to one file.
func (r *Repo) CommitDiff(ctx context.Context, c Commit, ch Change) ([]byte, error) {
	args := commitDiff(c, "-p", "--no-color", "--no-ext-diff", "-M")
	args = append(args, "--", r.spec(ch.Path))
	if ch.OrigPath != "" {
		args = append(args, r.spec(ch.OrigPath))
	}
	return r.run(ctx, args...)
}
