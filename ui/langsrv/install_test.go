package langsrv

import (
	"strings"
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
)

func waitInstall(t *testing.T, s *Service) (lines []string, done InstallEvent) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, ev := range s.DrainInstalls() {
			if ev.Done {
				return lines, ev
			}
			lines = append(lines, ev.Line)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("install did not finish")
	return nil, InstallEvent{}
}

func newTestService() *Service {
	return &Service{applied: map[string]domain.LanguageServerConfig{}, lastToast: map[string]time.Time{}}
}

func TestInstallStreamsOutputAndSucceeds(t *testing.T) {
	s := newTestService()
	// "sh" is the server command, so the post-install PATH check passes.
	l := Language{ID: "fake", Name: "Fake", Command: "sh"}
	inst := Installer{Tool: "sh", Args: []string{"-c", "echo fetching; echo linking >&2"}}
	if !s.Install(l, inst) {
		t.Fatal("Install refused to start")
	}
	if s.Install(l, inst) {
		t.Fatal("a second Install for the same language should be refused while one runs")
	}
	lines, done := waitInstall(t, s)
	if done.Err != nil {
		t.Fatalf("install failed: %v", done.Err)
	}
	got := strings.Join(lines, "|")
	for _, want := range []string{"$ sh -c", "fetching", "linking"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q lacks %q", got, want)
		}
	}
	if s.Installing("fake") {
		t.Fatal("still marked installing after Done")
	}
}

func TestInstallReportsFailureAndMissingServer(t *testing.T) {
	s := newTestService()
	l := Language{ID: "fake", Name: "Fake", Command: "sh"}
	s.Install(l, Installer{Tool: "sh", Args: []string{"-c", "echo 'EACCES: permission denied' >&2; exit 243"}})
	lines, done := waitInstall(t, s)
	if done.Err == nil || !strings.Contains(done.Err.Error(), "exit status 243") {
		t.Fatalf("err = %v, want exit status 243", done.Err)
	}
	if !strings.Contains(strings.Join(lines, "|"), "EACCES") {
		t.Fatalf("stderr not streamed: %q", lines)
	}

	// The installer succeeds but the server still can't be found.
	l2 := Language{ID: "fake2", Name: "Fake2", Command: "chapar-no-such-server"}
	s.Install(l2, Installer{Tool: "sh", Args: []string{"-c", "true"}})
	_, done = waitInstall(t, s)
	if done.Err == nil || !strings.Contains(done.Err.Error(), "still not on PATH") {
		t.Fatalf("err = %v, want a still-not-on-PATH error", done.Err)
	}
}
