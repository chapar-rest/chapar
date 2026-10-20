// Package gitx runs the system git binary for a workspace. Chapar does not
// bundle a git implementation: the user's git knows their credentials,
// hooks and config, and gives the same results as their terminal.
package gitx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrNotInstalled means no working git binary was found.
var ErrNotInstalled = errors.New("git is not installed")

// ErrNotRepo means the directory is not inside a git work tree.
var ErrNotRepo = errors.New("not a git repository")

// Git is a git binary that was found and runs.
type Git struct {
	path    string
	version string
}

// Find looks git up on PATH, then in the places installers put it, and
// checks that it runs. A GUI app may start with a shorter PATH than a
// shell, and on Windows it keeps the PATH it started with after git is
// installed. On macOS /usr/bin/git is a stub that fails until the command
// line tools are installed, so finding the file is not enough.
func Find(ctx context.Context) (*Git, error) {
	var paths []string
	if p, err := exec.LookPath("git"); err == nil {
		paths = append(paths, p)
	}
	paths = append(paths, knownPaths()...)
	for _, p := range paths {
		g := &Git{path: p}
		out, err := g.command(ctx, "", "--version").Output()
		if err != nil {
			continue
		}
		g.version = strings.TrimSpace(strings.TrimPrefix(string(out), "git version"))
		return g, nil
	}
	return nil, ErrNotInstalled
}

// knownPaths are where git installers put git, for when it is not on PATH.
func knownPaths() []string {
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{"/opt/homebrew/bin/git", "/usr/local/bin/git"}
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramW6432", "LocalAppData"} {
			base := os.Getenv(env)
			if base == "" {
				continue
			}
			if env == "LocalAppData" {
				base = filepath.Join(base, "Programs")
			}
			paths = append(paths, filepath.Join(base, "Git", "cmd", "git.exe"))
		}
	default:
		paths = []string{"/usr/bin/git", "/usr/local/bin/git"}
	}
	var found []string
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			found = append(found, p)
		}
	}
	return found
}

// Version is the version git reports, such as "2.53.0".
func (g *Git) Version() string { return g.version }

// command builds a git command that never prompts, never takes optional
// locks (a background status must not block the user's own git), and
// treats pathspecs as literal file names: workspace files often hold
// spaces and may hold glob characters.
func (g *Git) command(ctx context.Context, dir string, args ...string) *exec.Cmd {
	args = append([]string{"-c", "core.quotepath=false", "-c", "color.ui=false"}, args...)
	cmd := exec.CommandContext(ctx, g.path, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_LITERAL_PATHSPECS=1",
		"LC_ALL=C",
	)
	hideWindow(cmd)
	return cmd
}

// Repo is a workspace directory inside a git work tree. The work tree may
// be the workspace itself or a directory above it; paths Repo takes and
// returns are relative to the workspace and use forward slashes.
type Repo struct {
	git *Git
	// Dir is the workspace directory.
	Dir string
	// Root is the top of the work tree.
	Root string
	// prefix is Dir relative to Root, with a trailing slash, or "".
	prefix string
}

// Open finds the work tree dir belongs to. It returns ErrNotRepo when
// there is none.
func (g *Git) Open(ctx context.Context, dir string) (*Repo, error) {
	cmd := g.command(ctx, dir, "rev-parse", "--show-toplevel", "--show-prefix")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "not a git repository") {
			return nil, ErrNotRepo
		}
		return nil, runError([]string{"rev-parse"}, err, stderr.Bytes())
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) == 0 || lines[0] == "" {
		// Inside a .git directory, or a bare repository.
		return nil, ErrNotRepo
	}
	r := &Repo{git: g, Dir: dir, Root: filepath.FromSlash(lines[0])}
	if len(lines) > 1 {
		r.prefix = lines[1]
	}
	return r, nil
}

// Contained reports whether the work tree is the workspace itself rather
// than a directory above it.
func (r *Repo) Contained() bool { return r.prefix == "" }

// run runs git at the work tree root and returns its stdout.
func (r *Repo) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := r.git.command(ctx, r.Root, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, runError(args, err, stderr.Bytes())
	}
	return out, nil
}

// runCmd runs a command whose output nobody reads.
func runCmd(cmd *exec.Cmd, args []string) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return runError(args, err, stderr.Bytes())
	}
	return nil
}

// spec is the pathspec for a workspace-relative path, or the whole
// workspace for "".
func (r *Repo) spec(path string) string {
	if path == "" {
		if r.prefix == "" {
			return "."
		}
		return strings.TrimSuffix(r.prefix, "/")
	}
	return r.prefix + path
}

// rel turns a path git printed, relative to the root, into one relative
// to the workspace. ok is false for a path outside the workspace.
func (r *Repo) rel(path string) (string, bool) {
	if r.prefix == "" {
		return path, true
	}
	if !strings.HasPrefix(path, r.prefix) {
		return "", false
	}
	return path[len(r.prefix):], true
}

// ExitError is a git command that failed. Message is what git wrote to
// stderr.
type ExitError struct {
	Args    []string
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("git %s failed with exit code %d", e.Args[0], e.Code)
	}
	return "git " + e.Args[0] + ": " + e.Message
}

func runError(args []string, err error, stderr []byte) error {
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return err
	}
	msg := strings.TrimSpace(string(stderr))
	msg = strings.TrimPrefix(msg, "fatal: ")
	msg = strings.TrimPrefix(msg, "error: ")
	return &ExitError{Args: args, Code: exit.ExitCode(), Message: msg}
}
