package gitx

import (
	"bytes"
	"context"
	"strconv"
	"strings"
)

// ChangeKind is how a file differs from the last commit.
type ChangeKind int

const (
	Modified ChangeKind = iota
	Added
	Deleted
	Renamed
	Untracked
	Conflicted
)

func (k ChangeKind) String() string {
	switch k {
	case Added:
		return "Added"
	case Deleted:
		return "Deleted"
	case Renamed:
		return "Renamed"
	case Untracked:
		return "Untracked"
	case Conflicted:
		return "Conflicted"
	}
	return "Modified"
}

// Change is a file that differs from the last commit, staged or not.
type Change struct {
	// Path is relative to the workspace, with forward slashes.
	Path string
	// OrigPath is the path before a rename. It is "" when the file was
	// renamed from outside the workspace.
	OrigPath string
	Kind     ChangeKind
	// Staged and Unstaged are git's X and Y status letters, '.' for none.
	Staged, Unstaged byte
}

// Status is the state of the work tree and its branch.
type Status struct {
	// Branch is the checked out branch, or "" when HEAD is detached.
	Branch string
	// Head is the commit HEAD points at, or "" before the first commit.
	Head string
	// Upstream is the branch Branch tracks, such as "origin/main", or "".
	Upstream string
	// Ahead and Behind count commits against Upstream.
	Ahead, Behind int
	// Changes are the files in the workspace that differ from Head.
	Changes []Change
}

// Status reads the branch and the changed files in the workspace.
func (r *Repo) Status(ctx context.Context) (*Status, error) {
	out, err := r.run(ctx, "status", "--porcelain=v2", "-z", "--branch",
		"--untracked-files=all", "--", r.spec(""))
	if err != nil {
		return nil, err
	}
	return r.parseStatus(out), nil
}

// parseStatus reads `git status --porcelain=v2 -z --branch`.
func (r *Repo) parseStatus(out []byte) *Status {
	st := &Status{}
	recs := bytes.Split(out, []byte{0})
	for i := 0; i < len(recs); i++ {
		rec := string(recs[i])
		if rec == "" {
			continue
		}
		var c Change
		switch rec[0] {
		case '#':
			st.header(rec)
			continue
		case '1':
			f := strings.SplitN(rec, " ", 9)
			if len(f) < 9 {
				continue
			}
			if f[1] == "AD" {
				// Added to the index, then deleted: neither the last
				// commit nor the work tree has it.
				continue
			}
			c = change(f[1], f[8])
		case '2':
			f := strings.SplitN(rec, " ", 10)
			if len(f) < 10 {
				continue
			}
			c = change(f[1], f[9])
			// The path before the rename is the next record.
			if i+1 < len(recs) {
				i++
				c.OrigPath, _ = r.rel(string(recs[i]))
			}
			c.Kind = Renamed
		case 'u':
			f := strings.SplitN(rec, " ", 11)
			if len(f) < 11 {
				continue
			}
			c = change(f[1], f[10])
			c.Kind = Conflicted
		case '?':
			c = Change{Path: rec[2:], Kind: Untracked, Staged: '?', Unstaged: '?'}
		default:
			continue
		}
		p, ok := r.rel(c.Path)
		if !ok {
			continue
		}
		c.Path = p
		st.Changes = append(st.Changes, c)
	}
	return st
}

func (st *Status) header(rec string) {
	key, val, _ := strings.Cut(strings.TrimPrefix(rec, "# "), " ")
	switch key {
	case "branch.oid":
		if val != "(initial)" {
			st.Head = val
		}
	case "branch.head":
		if val != "(detached)" {
			st.Branch = val
		}
	case "branch.upstream":
		st.Upstream = val
	case "branch.ab":
		a, b, _ := strings.Cut(val, " ")
		st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(a, "+"))
		st.Behind, _ = strconv.Atoi(strings.TrimPrefix(b, "-"))
	}
}

func change(xy, path string) Change {
	c := Change{Path: path, Staged: xy[0], Unstaged: xy[1]}
	switch {
	case xy[0] == 'A' && xy[1] != 'D':
		c.Kind = Added
	case xy[0] == 'D' || xy[1] == 'D':
		c.Kind = Deleted
	default:
		c.Kind = Modified
	}
	return c
}
