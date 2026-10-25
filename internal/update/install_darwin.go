package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// bundlePath is the .app this process runs from.
func bundlePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return "", err
	}
	macos := filepath.Dir(exe)
	contents := filepath.Dir(macos)
	app := filepath.Dir(contents)
	if filepath.Base(macos) != "MacOS" || filepath.Base(contents) != "Contents" || filepath.Ext(app) != ".app" {
		return "", ErrUnsupported
	}
	return app, nil
}

// Detect works out how this copy updates. It runs codesign, so call it off
// the UI goroutine.
func Detect() Method {
	app, err := bundlePath()
	if err != nil {
		return Manual
	}
	if brewInstalled(app) {
		return Homebrew
	}
	// Run from the DMG, or translocated because it was never moved out of
	// Downloads: replacing it would not stick.
	if strings.HasPrefix(app, "/Volumes/") || strings.Contains(app, "/AppTranslocation/") {
		return Manual
	}
	if team, _ := teamID(app); team == "" {
		return Manual // unsigned dev build: nothing to check the update against
	}
	if !dirWritable(filepath.Dir(app)) {
		return Manual
	}
	return InApp
}

// brewInstalled reports whether app is the copy the Homebrew cask put in
// an Applications folder. Another copy elsewhere updates on its own.
func brewInstalled(app string) bool {
	home, _ := os.UserHomeDir()
	if dir := filepath.Dir(app); dir != "/Applications" && dir != filepath.Join(home, "Applications") {
		return false
	}
	for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
		if fi, err := os.Stat(filepath.Join(prefix, "Caskroom", "chapar")); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

// teamID is the Developer ID team that signed app, or "" if it is unsigned
// or ad-hoc signed.
func teamID(app string) (string, error) {
	out, err := exec.Command("codesign", "-dv", "--verbose=2", app).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codesign: %s", strings.TrimSpace(string(out)))
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if id, ok := strings.CutPrefix(line, "TeamIdentifier="); ok && id != "not set" {
			return strings.TrimSpace(id), nil
		}
	}
	return "", nil
}

// Install replaces the running app bundle with the one in the dmg at path,
// once its signature checks out and it was signed by the same team.
func Install(ctx context.Context, path string) error {
	app, err := bundlePath()
	if err != nil {
		return err
	}
	return installBundle(ctx, app, path)
}

func installBundle(ctx context.Context, app, path string) error {
	want, err := teamID(app)
	if err != nil || want == "" {
		return ErrUnsupported
	}

	mnt, err := os.MkdirTemp("", "chapar-update-mnt-")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(mnt) }()
	if out, err := exec.CommandContext(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen",
		"-mountpoint", mnt, path).CombinedOutput(); err != nil {
		return fmt.Errorf("opening the update: %s", strings.TrimSpace(string(out)))
	}
	defer func() { _ = exec.Command("hdiutil", "detach", "-force", mnt).Run() }()

	matches, _ := filepath.Glob(filepath.Join(mnt, "*.app"))
	if len(matches) != 1 {
		return errors.New("the update has no app in it")
	}
	newApp := matches[0]
	// Not --strict: the DMG layout leaves Finder info on the bundle folder,
	// which is outside the seal and fine with Gatekeeper.
	if out, err := exec.CommandContext(ctx, "codesign", "--verify", "--deep", newApp).CombinedOutput(); err != nil {
		return fmt.Errorf("the update's signature is invalid: %s", strings.TrimSpace(string(out)))
	}
	// Gatekeeper's own check: a notarized Developer ID app.
	if out, err := exec.CommandContext(ctx, "spctl", "--assess", "--type", "execute", newApp).CombinedOutput(); err != nil {
		return fmt.Errorf("the update fails the Gatekeeper check: %s", strings.TrimSpace(string(out)))
	}
	if got, err := teamID(newApp); err != nil || got != want {
		return fmt.Errorf("the update is signed by %q, not %q", got, want)
	}

	dir, base := filepath.Split(app)
	next := filepath.Join(dir, "."+base+".update")
	old := filepath.Join(dir, "."+base+".old")
	_ = os.RemoveAll(next)
	// ditto keeps the bundle's signature, extended attributes and links.
	if out, err := exec.CommandContext(ctx, "ditto", newApp, next).CombinedOutput(); err != nil {
		_ = os.RemoveAll(next)
		return fmt.Errorf("copying the update: %s", strings.TrimSpace(string(out)))
	}
	if err := swap(app, next, old); err != nil {
		_ = os.RemoveAll(next)
		return err
	}
	_ = os.RemoveAll(old)
	return nil
}

// Restart starts the installed app. The caller quits right after.
func Restart() error {
	app, err := bundlePath()
	if err != nil {
		return err
	}
	return exec.Command("open", "-n", app).Start()
}

// Cleanup removes what an earlier update left behind. Install removes the
// old bundle itself, so there is nothing to do on macOS.
func Cleanup() {}

// InstalledVersion is the version of the app bundle on disk, which differs
// from the running one after an update.
func InstalledVersion() string {
	app, err := bundlePath()
	if err != nil {
		return ""
	}
	return bundleVersion(app)
}

func bundleVersion(app string) string {
	out, err := exec.Command("plutil", "-extract", "CFBundleShortVersionString", "raw", "-o", "-",
		filepath.Join(app, "Contents", "Info.plist")).Output()
	if err != nil {
		return ""
	}
	return "v" + strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
}

// BrewUpgrade upgrades the Homebrew cask, writing brew's output to out.
func BrewUpgrade(ctx context.Context, out io.Writer) error {
	brew, err := exec.LookPath("brew")
	if err != nil {
		for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
			if _, serr := os.Stat(p); serr == nil {
				brew, err = p, nil
				break
			}
		}
	}
	if err != nil {
		return errors.New("brew was not found")
	}
	cmd := exec.CommandContext(ctx, brew, "upgrade", "--cask", "chapar")
	cmd.Stdout, cmd.Stderr = out, out
	cmd.Env = append(os.Environ(), "NONINTERACTIVE=1")
	return cmd.Run()
}
