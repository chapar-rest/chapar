package cookieui

import (
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
)

func TestParseExpiry(t *testing.T) {
	for _, s := range []string{"", " ", "Session"} {
		if got, err := parseExpiry(s); err != nil || !got.IsZero() {
			t.Errorf("parseExpiry(%q) = %v, %v; want session", s, got, err)
		}
	}
	got, err := parseExpiry("2026-10-01 12:30")
	want := time.Date(2026, 10, 1, 12, 30, 0, 0, time.Local)
	if err != nil || !got.Equal(want) {
		t.Errorf("parseExpiry = %v, %v; want %v", got, err, want)
	}
	if _, err := parseExpiry("tomorrow"); err == nil {
		t.Error("want error for unparseable expiry")
	}
}

func TestEditFlowSavesToStore(t *testing.T) {
	ws := t.TempDir()
	dir := func() (string, error) { return ws, nil }
	var errs []error
	d := New(Deps{
		Store:     cookies.NewStore(dir, nil),
		Envs:      func() []*domain.Environment { return nil },
		ActiveEnv: func() *domain.Environment { return nil },
		Error:     func(err error) { errs = append(errs, err) },
		Toast:     func(string) {},
	})
	d.load()

	d.add()
	d.edit.cookie.Name = "token"
	d.edit.cookie.Value = "abc"
	d.edit.expires = "bad"
	d.commitEdit()
	if d.edit.err == "" {
		t.Fatal("want an error for a bad expiry")
	}
	d.edit.expires = ""
	d.commitEdit()
	if d.edit.err == "" {
		t.Fatal("want an error for a missing domain")
	}
	d.edit.cookie.Domain = ".Example.com"
	d.commitEdit()
	if d.edit.err != "" || d.edit.isNew {
		t.Fatalf("commit failed: %q", d.edit.err)
	}
	if len(d.table.Rows) != 1 || d.table.Rows[0].Cells["domain"] != ".example.com" {
		t.Fatalf("rows = %+v", d.table.Rows)
	}

	// Saved to disk: a fresh store sees the cookie.
	jar, err := cookies.NewStore(dir, nil).For("")
	if err != nil {
		t.Fatal(err)
	}
	if all := jar.All(); len(all) != 1 || all[0].Value != "abc" || all[0].Source != domain.CookieSourceManual {
		t.Fatalf("persisted %+v", all)
	}

	d.deleteCookie(d.edit.cookie.ID)
	if d.edit != nil || len(d.table.Rows) != 0 {
		t.Fatalf("delete left edit=%v rows=%v", d.edit, d.table.Rows)
	}
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
}
