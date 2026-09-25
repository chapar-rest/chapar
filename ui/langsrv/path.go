package langsrv

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const pathMarker = "__CHAPAR_PATH__"

// FixPath extends this process's PATH so language servers installed by npm,
// Homebrew, pip, or version managers can be found — and can find their own
// runtimes (pyright-langserver is a `#!/usr/bin/env node` script).
//
// Apps started from Finder or the Dock inherit launchd's minimal PATH
// (/usr/bin:/bin:/usr/sbin:/sbin), not the user's shell PATH. When not started
// from a terminal, FixPath asks the user's login shell for its PATH, the way
// editors like VS Code do, then appends common install directories. It can
// take a moment and belongs on a background goroutine.
func FixPath() {
	if runtime.GOOS == "windows" {
		return
	}
	dirs := filepath.SplitList(os.Getenv("PATH"))
	if os.Getenv("TERM") == "" {
		dirs = appendNew(dirs, shellPath()...)
	}
	home, _ := os.UserHomeDir()
	for _, d := range []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		filepath.Join(home, ".local/bin"),
		filepath.Join(home, ".npm-global/bin"),
		filepath.Join(home, ".volta/bin"),
		filepath.Join(home, ".bun/bin"),
		filepath.Join(home, "go/bin"),
		filepath.Join(home, ".dotnet/tools"),
	} {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			dirs = appendNew(dirs, d)
		}
	}
	// Per-user bins of `pip install --user` (macOS) and `gem install
	// --user-install`, which are versioned directories.
	for _, pattern := range []string{
		filepath.Join(home, "Library/Python/*/bin"),
		filepath.Join(home, ".gem/ruby/*/bin"),
	} {
		matches, _ := filepath.Glob(pattern)
		dirs = appendNew(dirs, matches...)
	}
	_ = os.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

// shellPath returns the PATH an interactive login shell would have.
func shellPath() []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
		if runtime.GOOS == "darwin" {
			shell = "/bin/zsh"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Markers fence the value off from anything rc files print.
	out, err := exec.CommandContext(ctx, shell, "-ilc",
		`printf '%s%s%s' "`+pathMarker+`" "$PATH" "`+pathMarker+`"`).Output()
	if err != nil {
		return nil
	}
	parts := strings.Split(string(out), pathMarker)
	if len(parts) < 3 {
		return nil
	}
	return filepath.SplitList(parts[len(parts)-2])
}

func appendNew(dirs []string, add ...string) []string {
	for _, d := range add {
		if d != "" && !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}
