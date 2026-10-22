package gitx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Environment variables that pass credentials to the askpass helper. The
// helper is the chapar binary itself, run by git with AskPassEnv set.
const (
	AskPassEnv   = "CHAPAR_GIT_ASKPASS"
	askUserEnv   = "CHAPAR_GIT_USERNAME"
	askSecretEnv = "CHAPAR_GIT_PASSWORD"
)

// ErrAuth means the remote refused the credentials, or git needed some and
// had none.
var ErrAuth = errors.New("the remote needs a user name and token")

// ErrRejected means the remote has commits the local branch lacks.
var ErrRejected = errors.New("the remote has commits this branch doesn't have; pull first")

// ErrDiverged means a pull can't fast-forward: both sides have new commits.
var ErrDiverged = errors.New("this branch and the remote both have new commits")

// ErrNoUpstream means the branch doesn't track a remote branch and the
// remote has no branch of the same name.
var ErrNoUpstream = errors.New("the remote has no branch to pull from")

// Auth is a user name and token (or password) for an HTTPS remote. The
// zero Auth leaves authentication to the user's own git setup: credential
// helpers, SSH keys and agents.
type Auth struct {
	Username string
	Password string
}

// remoteCommand is a command that talks to a remote. git asks the chapar
// binary for credentials instead of a terminal; without auth, the user's
// credential helpers still answer first. SSH runs in batch mode, so a missing key or
// unknown host fails instead of waiting for a prompt nobody sees.
func (g *Git) remoteCommand(ctx context.Context, dir string, auth Auth, args ...string) *exec.Cmd {
	if auth.Username != "" || auth.Password != "" {
		// The user gave Chapar a token for this server: use it rather than
		// what a credential helper has, which may be what just failed.
		args = append([]string{"-c", "credential.helper="}, args...)
	}
	cmd := g.command(ctx, dir, args...)
	if exe, err := os.Executable(); err == nil {
		cmd.Env = append(cmd.Env, "GIT_ASKPASS="+exe, AskPassEnv+"=1")
	}
	if auth.Username != "" || auth.Password != "" {
		cmd.Env = append(cmd.Env, askUserEnv+"="+auth.Username, askSecretEnv+"="+auth.Password)
	}
	if os.Getenv("GIT_SSH_COMMAND") == "" && !g.hasSSHCommand(ctx, dir) {
		cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	return cmd
}

// hasSSHCommand reports whether git config sets core.sshCommand, which
// GIT_SSH_COMMAND would override.
func (g *Git) hasSSHCommand(ctx context.Context, dir string) bool {
	out, _ := g.command(ctx, dir, "config", "--get", "core.sshCommand").Output()
	return strings.TrimSpace(string(out)) != ""
}

func (r *Repo) runRemote(ctx context.Context, auth Auth, args ...string) error {
	return remoteError(runCmd(r.git.remoteCommand(ctx, r.Root, auth, args...), args))
}

// remoteError turns the failures a user can act on into the errors above.
func remoteError(err error) error {
	var ex *ExitError
	if !errors.As(err, &ex) {
		return err
	}
	msg := strings.ToLower(ex.Message)
	switch {
	case strings.Contains(msg, "authentication failed"),
		strings.Contains(msg, "could not read username"),
		strings.Contains(msg, "could not read password"),
		strings.Contains(msg, "terminal prompts disabled"),
		strings.Contains(msg, "invalid username or password"),
		strings.Contains(msg, "the requested url returned error: 401"),
		strings.Contains(msg, "the requested url returned error: 403"):
		return fmt.Errorf("%w: %s", ErrAuth, ex.Message)
	case strings.Contains(msg, "[rejected]"), strings.Contains(msg, "non-fast-forward"),
		strings.Contains(msg, "fetch first"):
		return ErrRejected
	case strings.Contains(msg, "not possible to fast-forward"), strings.Contains(msg, "diverging branches"):
		return ErrDiverged
	case strings.Contains(msg, "permission denied (publickey"), strings.Contains(msg, "host key verification failed"):
		return fmt.Errorf("%s. Chapar connects with your SSH keys and agent; check that ssh to the host works in a terminal", ex.Message)
	}
	return err
}

// Clone copies the repository at url into dir, which must not exist or
// be empty.
func (g *Git) Clone(ctx context.Context, url, dir string, auth Auth) error {
	args := []string{"clone", "--quiet", "--", url, dir}
	return remoteError(runCmd(g.remoteCommand(ctx, "", auth, args...), args))
}

// Fetch updates the remote branches, so ahead and behind counts are
// current.
func (r *Repo) Fetch(ctx context.Context, auth Auth) error {
	name, _, err := r.Remote(ctx)
	if err != nil || name == "" {
		return err
	}
	return r.runRemote(ctx, auth, "fetch", "--quiet", "--prune", name)
}

// Pull brings in the remote branch's new commits when the local branch has
// none of its own. A branch with no upstream starts tracking the remote's
// branch of the same name. It reports whether HEAD moved.
func (r *Repo) Pull(ctx context.Context, auth Auth, st *Status) (bool, error) {
	if st.Branch == "" {
		return false, errors.New("HEAD is detached; check out a branch to pull")
	}
	name, _, err := r.Remote(ctx)
	if err != nil {
		return false, err
	}
	if name == "" {
		return false, errors.New("the repository has no remote")
	}
	if st.Upstream == "" {
		if err := r.Fetch(ctx, auth); err != nil {
			return false, err
		}
		ref := name + "/" + st.Branch
		if _, err := r.run(ctx, "rev-parse", "--verify", "--quiet", "refs/remotes/"+ref); err != nil {
			return false, ErrNoUpstream
		}
		if st.Head == "" {
			// No commits to keep: start the branch at the remote's. git
			// refuses if that would overwrite an untracked file.
			_, err := r.run(ctx, "checkout", "--quiet", "-B", st.Branch, "--track", ref)
			return err == nil, err
		}
		if _, err := r.run(ctx, "branch", "--quiet", "--set-upstream-to="+ref); err != nil {
			return false, err
		}
	}
	before := st.Head
	if err := r.runRemote(ctx, auth, "pull", "--quiet", "--ff-only", "--no-rebase"); err != nil {
		return false, err
	}
	out, err := r.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != before, nil
}

// Push sends the branch's commits to the remote. A branch with no
// upstream is pushed to the remote branch of the same name and tracks it.
func (r *Repo) Push(ctx context.Context, auth Auth, st *Status) error {
	if st.Branch == "" {
		return errors.New("HEAD is detached; check out a branch to push")
	}
	if st.Head == "" {
		return errors.New("there are no commits to push")
	}
	if st.Upstream != "" {
		return r.runRemote(ctx, auth, "push", "--quiet")
	}
	name, _, err := r.Remote(ctx)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("the repository has no remote")
	}
	return r.runRemote(ctx, auth, "push", "--quiet", "--set-upstream", name, "HEAD:refs/heads/"+st.Branch)
}

// AskPass answers a git credential prompt from the environment: the user
// name for a "Username for ..." prompt, the token for a "Password for ..."
// one. It returns the exit code; git treats a non-zero code as no answer.
func AskPass(args []string, out io.Writer) int {
	prompt := strings.ToLower(strings.Join(args, " "))
	var v string
	switch {
	case strings.HasPrefix(prompt, "username"):
		v = os.Getenv(askUserEnv)
	case strings.HasPrefix(prompt, "password"):
		v = os.Getenv(askSecretEnv)
	}
	if v == "" {
		return 1
	}
	_, _ = fmt.Fprintln(out, v)
	return 0
}
