package ui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/update"
	"github.com/chapar-rest/chapar/ui/langsrv"
	"github.com/chapar-rest/chapar/version"
)

const updateToastID = "app-update"

// restartAfterExit makes Run start the updated app once the window closes.
var restartAfterExit atomic.Bool

type updatePhase int

const (
	updateNone updatePhase = iota
	updateAvailable
	updateDownloading
	updateInstalling
	updateInstalled
	updateFailed
)

// updater checks for a new release once at startup and installs it on
// request. Its goroutines write under mu and wake the UI; reportUpdate
// shows the state from the frame, and is the only place that logs, since the
// console logger is not safe across goroutines.
type updater struct {
	mu          sync.Mutex
	phase       updatePhase
	method      update.Method
	release     update.Release
	asset       string // release file for this platform, "" if none
	done, total int64
	err         error
	logs        []string
	cancel      context.CancelFunc

	shown    updatePhase // last phase shown
	shownPct int
}

type updateState struct {
	phase       updatePhase
	method      update.Method
	release     update.Release
	asset       string
	done, total int64
	err         error
}

func (u *updater) state() updateState {
	u.mu.Lock()
	defer u.mu.Unlock()
	return updateState{u.phase, u.method, u.release, u.asset, u.done, u.total, u.err}
}

func (u *updater) set(f func()) {
	u.mu.Lock()
	f()
	u.mu.Unlock()
}

func (u *updater) log(line string) {
	u.set(func() { u.logs = append(u.logs, line) })
}

func updateClient() update.Client {
	return update.Client{UserAgent: version.GetAgentName()}
}

// startUpdateCheck looks for a newer release in the background. It runs on
// the first frame, so wake is ready for the result.
func (a *App) startUpdateCheck(wake func()) {
	go update.Cleanup()
	cur := version.GetAppVersion()
	if !prefs.GetGlobalConfig().Spec.General.CheckForUpdates || !update.IsRelease(cur) {
		return
	}
	u := a.upd
	go func() {
		r, err := updateClient().Latest(context.Background())
		if err != nil {
			u.log("Update check failed: " + err.Error())
			wake()
			return
		}
		if !update.Newer(cur, r.Version) {
			return
		}
		method := update.Detect()
		name, _ := update.AssetName(r.Version, runtime.GOOS, runtime.GOARCH)
		if _, ok := r.Asset(name); !ok {
			name = ""
		}
		// In-app installs need a signed asset; releases from before signing
		// are downloaded by hand.
		if _, ok := r.Asset(name + update.SigSuffix); method == update.InApp && (name == "" || !ok) {
			method = update.Manual
		}
		u.set(func() {
			u.phase, u.method, u.release, u.asset = updateAvailable, method, r, name
		})
		wake()
	}()
}

// reportUpdate shows the update toast when the state changed.
func (a *App) reportUpdate() {
	u := a.upd
	u.mu.Lock()
	logs := u.logs
	u.logs = nil
	u.mu.Unlock()
	for _, l := range logs {
		logger.Info(l)
	}

	s := u.state()
	pct := 0
	if s.total > 0 {
		pct = int(s.done * 100 / s.total)
	}
	if s.phase == u.shown && pct == u.shownPct {
		return
	}
	host := a.toasts()
	if host == nil {
		return
	}
	u.shown, u.shownPct = s.phase, pct

	r := s.release
	whatsNew := ui.ToastAction{Label: "See what's new", KeepOpen: true, OnClick: func() { openUpdateURL(r.URL) }}
	download := ui.ToastAction{Label: "Download", Primary: true, OnClick: func() { openUpdateURL(s.downloadURL()) }}

	switch s.phase {
	case updateAvailable:
		opts := ui.ToastOpts{
			ID:      updateToastID,
			Title:   "Chapar " + r.Version + " is available",
			Message: "You have " + version.GetAppVersion() + ".",
			Variant: ui.ToastInfo,
		}
		switch s.method {
		case update.InApp:
			opts.Actions = []ui.ToastAction{
				{Label: "Install latest", Primary: true, KeepOpen: true, OnClick: a.installUpdate},
				whatsNew,
			}
		case update.Homebrew:
			opts.Message += " Chapar was installed with Homebrew, so this runs brew upgrade --cask chapar."
			opts.Actions = []ui.ToastAction{
				{Label: "Install latest", Primary: true, KeepOpen: true, OnClick: a.installUpdate},
				whatsNew,
			}
		default:
			opts.Message += " " + manualUpdateHint()
			opts.Actions = []ui.ToastAction{download, whatsNew}
		}
		host.Notify(opts)
	case updateDownloading:
		msg := "Starting…"
		if s.total > 0 {
			msg = fmt.Sprintf("%d%% of %.1f MB", pct, float64(s.total)/(1<<20))
		}
		host.Notify(ui.ToastOpts{
			ID:        updateToastID,
			Title:     "Downloading Chapar " + r.Version + "…",
			Message:   msg,
			Variant:   ui.ToastInfo,
			Actions:   []ui.ToastAction{{Label: "Cancel", OnClick: a.cancelUpdate}},
			OnDismiss: a.cancelUpdate,
		})
	case updateInstalling:
		opts := ui.ToastOpts{
			ID:      updateToastID,
			Title:   "Installing Chapar " + r.Version + "…",
			Message: "This takes a few seconds.",
			Variant: ui.ToastInfo,
		}
		if s.method == update.Homebrew {
			opts.Message = "Running brew upgrade --cask chapar. The output is in the Console."
			opts.Actions = []ui.ToastAction{{Label: "Show console", KeepOpen: true, OnClick: a.console.Show}}
		}
		host.Notify(opts)
	case updateInstalled:
		a.notifs.Add("Chapar "+r.Version+" is installed", ui.ToastSuccess)
		host.Notify(ui.ToastOpts{
			ID:      updateToastID,
			Title:   "Chapar " + r.Version + " is installed",
			Message: "Restart Chapar to start using it.",
			Variant: ui.ToastSuccess,
			Actions: []ui.ToastAction{
				{Label: "Restart now", Primary: true, OnClick: a.restartForUpdate},
				{Label: "Later"},
			},
		})
	case updateFailed:
		if errors.Is(s.err, context.Canceled) {
			host.Dismiss(updateToastID)
			return
		}
		logger.Error("Updating Chapar failed: " + s.err.Error())
		a.notifs.Add("Updating Chapar failed; see the Console", ui.ToastError)
		host.Notify(ui.ToastOpts{
			ID:      updateToastID,
			Title:   "Couldn't update Chapar",
			Message: s.err.Error() + "\nYou can download " + r.Version + " and install it yourself.",
			Variant: ui.ToastError,
			Actions: []ui.ToastAction{download, {Label: "Show console", KeepOpen: true, OnClick: a.console.Show}},
		})
	}
}

// downloadURL is this platform's release file, or the release page when the
// release has none.
func (s updateState) downloadURL() string {
	if a, ok := s.release.Asset(s.asset); ok {
		return a.URL
	}
	return s.release.URL
}

func manualUpdateHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "Download it, then drag Chapar to Applications to replace this copy."
	case "windows":
		return "Download it, then replace Chapar.exe with the one in the zip."
	}
	return "Download it, then run install-linux.sh from the archive."
}

func openUpdateURL(url string) {
	if err := langsrv.OpenURL(url); err != nil {
		logger.Error(fmt.Sprintf("opening %s: %v", url, err))
	}
}

// installUpdate downloads, verifies and installs the new release, or has
// Homebrew upgrade the cask.
func (a *App) installUpdate() {
	u := a.upd
	ctx, cancel := context.WithCancel(context.Background())
	s := u.state()
	if s.phase != updateAvailable {
		cancel()
		return
	}
	next := updateDownloading
	if s.method == update.Homebrew {
		next = updateInstalling
	}
	u.set(func() { u.phase, u.cancel, u.done, u.total = next, cancel, 0, 0 })
	wake := a.wake

	go func() {
		defer cancel()
		var err error
		if s.method == update.Homebrew {
			err = brewUpgrade(ctx, u, s.release.Version)
		} else {
			err = downloadAndInstall(ctx, u, s, wake)
		}
		u.set(func() {
			u.cancel = nil
			if err != nil {
				u.phase, u.err = updateFailed, err
			} else {
				u.phase = updateInstalled
			}
		})
		wake()
	}()
}

func downloadAndInstall(ctx context.Context, u *updater, s updateState, wake func()) error {
	path, err := updateClient().Download(ctx, s.release, s.asset, func(done, total int64) {
		u.mu.Lock()
		before := pctOf(u.done, u.total)
		u.done, u.total = done, total
		changed := pctOf(done, total) != before
		u.mu.Unlock()
		if changed {
			wake()
		}
	})
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(path) }()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	u.set(func() { u.phase = updateInstalling })
	wake()
	return update.Install(ctx, path)
}

func pctOf(done, total int64) int64 {
	if total <= 0 {
		return 0
	}
	return done * 100 / total
}

func brewUpgrade(ctx context.Context, u *updater, want string) error {
	pr, pw := io.Pipe()
	logged := make(chan struct{})
	go func() {
		defer close(logged)
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			u.log("brew: " + sc.Text())
		}
		_, _ = io.Copy(io.Discard, pr)
	}()
	err := update.BrewUpgrade(ctx, pw)
	_ = pw.Close()
	<-logged
	if err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}
	// The tap is bumped after the release builds, so brew can be a little
	// behind GitHub.
	if got := update.InstalledVersion(); !strings.EqualFold(got, want) {
		return fmt.Errorf("the Homebrew cask is not at %s yet; try again later", want)
	}
	return nil
}

func (a *App) cancelUpdate() {
	u := a.upd
	u.mu.Lock()
	cancel := u.cancel
	u.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// restartForUpdate closes the window; Run then starts the new version.
func (a *App) restartForUpdate() {
	quit := func() {
		restartAfterExit.Store(true)
		if a.uiCtx != nil && a.uiCtx.Window() != nil {
			a.uiCtx.Window().Close()
		}
		a.invalidate() // the loop notices the close on its next frame
	}
	if a.ws != nil && a.ws.HasDirty() {
		a.confirmClose("Restart Chapar?", "Restarting will close unsaved tabs. Continue?", quit)
		return
	}
	quit()
}
