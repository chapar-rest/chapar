package update

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallBundle swaps a real signed release for a newer one. It needs two
// release DMGs, so it only runs when CHAPAR_UPDATE_TEST_DMGS is "old.dmg:new.dmg".
func TestInstallBundle(t *testing.T) {
	oldDMG, newDMG, ok := strings.Cut(os.Getenv("CHAPAR_UPDATE_TEST_DMGS"), ":")
	if !ok {
		t.Skip("CHAPAR_UPDATE_TEST_DMGS not set")
	}
	dir := t.TempDir()
	mnt := filepath.Join(t.TempDir(), "mnt")
	if out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-readonly", "-mountpoint", mnt, oldDMG).CombinedOutput(); err != nil {
		t.Fatalf("attach: %s", out)
	}
	app := filepath.Join(dir, "Chapar.app")
	out, err := exec.Command("ditto", filepath.Join(mnt, "Chapar.app"), app).CombinedOutput()
	_ = exec.Command("hdiutil", "detach", "-force", mnt).Run()
	if err != nil {
		t.Fatalf("ditto: %s", out)
	}
	before := bundleVersion(app)

	if err := installBundle(context.Background(), app, newDMG); err != nil {
		t.Fatal(err)
	}
	after := bundleVersion(app)
	if after == before || after == "v" {
		t.Fatalf("version %s -> %s", before, after)
	}
	if out, err := exec.Command("codesign", "--verify", "--deep", app).CombinedOutput(); err != nil {
		t.Fatalf("installed app fails codesign: %s", out)
	}
	left, _ := filepath.Glob(filepath.Join(dir, ".*"))
	if len(left) != 0 {
		t.Fatalf("left behind %v", left)
	}
	t.Logf("%s -> %s", before, after)

	// An unsigned bundle can't be checked against, so it is never replaced.
	if out, err := exec.Command("codesign", "--remove-signature", filepath.Join(app, "Contents", "MacOS", "chapar")).CombinedOutput(); err != nil {
		t.Fatalf("remove signature: %s", out)
	}
	if err := installBundle(context.Background(), app, newDMG); err != ErrUnsupported {
		t.Fatalf("unsigned install: %v", err)
	}
}
