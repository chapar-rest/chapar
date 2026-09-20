package cookies

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func names(cs []*http.Cookie) string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return strings.Join(out, ",")
}

func newTestJar(now time.Time) *Jar {
	j := NewJar(nil)
	j.now = func() time.Time { return now }
	return j
}

func TestSetCookieDomainRules(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		url     string
		cookie  http.Cookie
		outcome Outcome
		reason  string
	}{
		{"host only", "https://api.example.com/x", http.Cookie{Name: "a", Value: "1"}, OutcomeNew, ""},
		{"parent domain", "https://api.example.com/x", http.Cookie{Name: "a", Value: "1", Domain: ".example.com"}, OutcomeNew, ""},
		{"other domain", "https://api.example.com/x", http.Cookie{Name: "a", Value: "1", Domain: "evil.com"}, OutcomeRejected, "does not match"},
		{"public suffix", "https://example.co.uk/", http.Cookie{Name: "a", Value: "1", Domain: "co.uk"}, OutcomeRejected, "public suffix"},
		{"secure over http", "http://example.com/", http.Cookie{Name: "a", Value: "1", Secure: true}, OutcomeRejected, "Secure"},
		{"secure over http on localhost", "http://localhost:8080/", http.Cookie{Name: "a", Value: "1", Secure: true}, OutcomeNew, ""},
		{"domain on ip", "http://127.0.0.1/", http.Cookie{Name: "a", Value: "1", Domain: "0.0.1"}, OutcomeRejected, "does not match"},
		{"expired and unknown", "https://example.com/", http.Cookie{Name: "a", Value: "1", MaxAge: -1}, OutcomeRejected, "already expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newTestJar(now)
			ev := j.set(mustURL(t, tt.url), &tt.cookie)
			if ev.Outcome != tt.outcome {
				t.Fatalf("outcome = %s (%s), want %s", ev.Outcome, ev.Reason, tt.outcome)
			}
			if !strings.Contains(ev.Reason, tt.reason) {
				t.Fatalf("reason = %q, want it to contain %q", ev.Reason, tt.reason)
			}
		})
	}
}

func TestCookiesMatching(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	j := newTestJar(now)
	origin := mustURL(t, "https://api.example.com/v1/users/me")

	j.SetCookies(origin, []*http.Cookie{
		{Name: "host", Value: "1"},                                   // host-only, path /v1/users
		{Name: "wide", Value: "1", Domain: "example.com", Path: "/"}, // all subdomains
		{Name: "v1", Value: "1", Path: "/v1"},
		{Name: "sec", Value: "1", Path: "/", Secure: true},
	})

	tests := []struct {
		url  string
		want string
	}{
		{"https://api.example.com/v1/users/42", "host,v1,sec,wide"},
		{"https://api.example.com/v1", "v1,sec,wide"},
		{"https://api.example.com/v10", "sec,wide"},
		{"http://api.example.com/v1/users/1", "host,v1,wide"},
		{"https://www.example.com/v1/users/1", "wide"},
		{"https://example.org/", ""},
	}
	for _, tt := range tests {
		got := names(j.Cookies(mustURL(t, tt.url)))
		// sec and wide share the / path; order between them is by creation,
		// and they were created at the same instant, so compare as sets there.
		if !sameSet(got, tt.want) {
			t.Errorf("%s: got %q, want %q", tt.url, got, tt.want)
		}
	}

	if got := names(j.Cookies(mustURL(t, "https://api.example.com/v1/users/42"))); !strings.HasPrefix(got, "host,v1,") {
		t.Errorf("longer paths should come first, got %q", got)
	}
}

func sameSet(a, b string) bool {
	as, bs := strings.Split(a, ","), strings.Split(b, ",")
	if len(as) != len(bs) {
		return false
	}
	seen := map[string]int{}
	for _, s := range as {
		seen[s]++
	}
	for _, s := range bs {
		seen[s]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func TestExpiryUpdateAndDelete(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	j := newTestJar(now)
	u := mustURL(t, "https://example.com/")

	first := j.set(u, &http.Cookie{Name: "s", Value: "1", MaxAge: 60})
	if first.Outcome != OutcomeNew || !first.Cookie.Expires.Equal(now.Add(time.Minute)) {
		t.Fatalf("unexpected first event %+v", first)
	}

	// A disabled cookie stays disabled when the server updates it.
	c := j.All()[0]
	c.Enabled = false
	if _, err := j.Upsert(c); err != nil {
		t.Fatal(err)
	}
	upd := j.set(u, &http.Cookie{Name: "s", Value: "2", Expires: now.Add(time.Hour)})
	if upd.Outcome != OutcomeUpdated || upd.Cookie.ID != first.Cookie.ID || upd.Cookie.Enabled {
		t.Fatalf("unexpected update event %+v", upd)
	}
	if got := j.Cookies(u); len(got) != 0 {
		t.Fatalf("disabled cookie was sent: %v", got)
	}

	del := j.set(u, &http.Cookie{Name: "s", MaxAge: -1})
	if del.Outcome != OutcomeDeleted || len(j.All()) != 0 {
		t.Fatalf("unexpected delete event %+v, jar %v", del, j.All())
	}

	// Cookies that expire on their own are dropped when read.
	j.set(u, &http.Cookie{Name: "t", Value: "1", MaxAge: 1})
	j.now = func() time.Time { return now.Add(2 * time.Second) }
	if got := j.Cookies(u); len(got) != 0 || len(j.All()) != 0 {
		t.Fatalf("expired cookie kept: %v", j.All())
	}
}

func TestUpsertAndClear(t *testing.T) {
	j := NewJar(nil)
	if _, err := j.Upsert(domain.Cookie{Name: "x"}); err == nil {
		t.Fatal("want error for cookie without a domain")
	}

	c, err := j.Upsert(domain.Cookie{Name: "a", Value: "1", Domain: ".Example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if c.Domain != "example.com" || c.Path != "/" || c.Source != domain.CookieSourceManual || c.ID == "" {
		t.Fatalf("unexpected normalized cookie %+v", c)
	}

	// Renaming through Upsert replaces the old entry.
	c.Name = "b"
	if _, err := j.Upsert(c); err != nil {
		t.Fatal(err)
	}
	if all := j.All(); len(all) != 1 || all[0].Name != "b" {
		t.Fatalf("rename left %+v", all)
	}

	if _, err := j.Upsert(domain.Cookie{Name: "c", Domain: "other.com", Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	j.ClearSession()
	if all := j.All(); len(all) != 1 || all[0].Name != "c" {
		t.Fatalf("ClearSession left %+v", all)
	}
	j.ClearDomain("other.com")
	if len(j.All()) != 0 {
		t.Fatalf("ClearDomain left %+v", j.All())
	}
}

func TestRecorderSkipsExplicitCookies(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
			http.Redirect(w, r, "/home", http.StatusFound)
		case "/home":
			gotCookie = r.Header.Get("Cookie")
		}
	}))
	defer srv.Close()

	ws := t.TempDir()
	store := NewStore(func() (string, error) { return ws, nil }, nil)

	send := func(path, header string) *Recorder {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if header != "" {
			req.Header.Set("Cookie", header)
		}
		client := &http.Client{}
		rec, err := store.Attach("env", client, req)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		return rec
	}

	// The cookie set on the redirect hop is sent to the redirect target.
	rec := send("/login", "")
	if gotCookie != "session=abc" {
		t.Fatalf("redirect target got Cookie %q", gotCookie)
	}
	if ev := rec.Events(); len(ev) != 1 || ev[0].Outcome != OutcomeNew {
		t.Fatalf("events = %+v", ev)
	}

	// A Cookie header typed on the request wins over the jar.
	rec = send("/home", "session=mine")
	if gotCookie != "session=mine" {
		t.Fatalf("got Cookie %q, want only the explicit one", gotCookie)
	}
	if len(rec.Sent()) != 0 {
		t.Fatalf("jar sent %v", rec.Sent())
	}

	// Without the explicit header the jar cookie is sent again.
	rec = send("/home", "")
	if gotCookie != "session=abc" || names(rec.Sent()) != "session" {
		t.Fatalf("got Cookie %q, sent %v", gotCookie, rec.Sent())
	}

	var nilStore *Store
	if rec, err := nilStore.Attach("env", &http.Client{}, &http.Request{}); rec != nil || err != nil {
		t.Fatal("nil store should attach nothing")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	ws := t.TempDir()
	dir := func() (string, error) { return ws, nil }

	s := NewStore(dir, nil)
	j, err := s.For("env1")
	if err != nil {
		t.Fatal(err)
	}
	j.SetCookies(mustURL(t, "https://example.com/"), []*http.Cookie{{Name: "a", Value: "1", MaxAge: 3600}})
	if err := s.Save("env1"); err != nil {
		t.Fatal(err)
	}

	file := File(ws, "env1")
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("jar file mode = %v, want 0600", info.Mode().Perm())
	}
	ignore, err := os.ReadFile(filepath.Join(ws, StateDir, ".gitignore"))
	if err != nil || string(ignore) != "*\n" {
		t.Errorf(".gitignore = %q, %v", ignore, err)
	}

	loaded, err := NewStore(dir, nil).For("env1")
	if err != nil {
		t.Fatal(err)
	}
	all := loaded.All()
	if len(all) != 1 || all[0].Name != "a" || all[0].Value != "1" || all[0].Domain != "example.com" {
		t.Fatalf("loaded %+v", all)
	}

	// Jars are separate per environment.
	other, err := s.For("")
	if err != nil {
		t.Fatal(err)
	}
	if len(other.All()) != 0 {
		t.Fatal("no-environment jar should be empty")
	}

	if err := s.Delete("env1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("jar file still exists: %v", err)
	}
}
