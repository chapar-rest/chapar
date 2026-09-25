package langsrv

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// installTimeout bounds one install; package managers can be slow, but a
// hung one must not keep "Installing…" up forever.
const installTimeout = 10 * time.Minute

// Installer is a command that installs a language server.
type Installer struct {
	Tool string // executable that must be on PATH (npm, brew, ...)
	Args []string
}

func (i Installer) String() string {
	return i.Tool + " " + JoinArgs(i.Args)
}

// Installer returns the preferred installer whose tool is on PATH.
func (l Language) Installer() (Installer, bool) {
	for _, i := range l.Installers {
		if _, err := exec.LookPath(i.Tool); err == nil {
			return i, true
		}
	}
	return Installer{}, false
}

// InstallEvent reports install progress: output lines while it runs, then
// one event with Done set.
type InstallEvent struct {
	Language  Language
	Installer Installer
	Line      string
	Done      bool
	Err       error // with Done: nil on success
}

// Install runs inst for l in the background. Progress arrives through
// DrainInstalls. It returns false if an install for l is already running.
func (s *Service) Install(l Language, inst Installer) bool {
	s.mu.Lock()
	if s.installing == nil {
		s.installing = map[string]bool{}
	}
	if s.installing[l.ID] {
		s.mu.Unlock()
		return false
	}
	s.installing[l.ID] = true
	s.mu.Unlock()
	command := l.Command
	if c, ok := s.applied[l.ID]; ok && c.Command != "" {
		command = expandHome(c.Command)
	}
	go s.runInstall(l, inst, command)
	return true
}

// Installing reports whether an install for the language is running.
func (s *Service) Installing(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.installing[id]
}

// runInstall runs inst, then checks that command (the server the settings
// launch) can now be found.
func (s *Service) runInstall(l Language, inst Installer, command string) {
	emit := func(ev InstallEvent) {
		ev.Language, ev.Installer = l, inst
		s.mu.Lock()
		s.installs = append(s.installs, ev)
		if ev.Done {
			delete(s.installing, l.ID)
			// The install (and the FixPath below it) may have put the server
			// on PATH; drop what the settings page resolved before that.
			clear(s.lookups)
		}
		s.mu.Unlock()
		s.poke()
	}

	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, inst.Tool, inst.Args...)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	emit(InstallEvent{Line: "$ " + inst.String()})
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		emit(InstallEvent{Done: true, Err: err})
		return
	}
	lines := make(chan struct{})
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			if line := strings.TrimRight(sc.Text(), " \r"); line != "" {
				emit(InstallEvent{Line: line})
			}
		}
		_, _ = io.Copy(io.Discard, pr)
	}()
	err := cmd.Wait()
	_ = pw.Close()
	<-lines
	if ctx.Err() != nil {
		err = fmt.Errorf("timed out after %v", installTimeout)
	}
	if err == nil {
		// The tool may have installed into a directory the app's PATH lacks
		// (pip's user bin, for one).
		FixPath()
		if _, lerr := exec.LookPath(command); lerr != nil {
			err = fmt.Errorf("installed, but %s is still not on PATH; set its full path in Settings → Language servers", command)
		}
	}
	emit(InstallEvent{Done: true, Err: err})
}

// DrainInstalls returns the install events since the last call.
func (s *Service) DrainInstalls() []InstallEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs := s.installs
	s.installs = nil
	return evs
}

// OpenURL opens url in the default browser.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
