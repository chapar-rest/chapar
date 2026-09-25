// Package cookies keeps per-environment cookie jars that follow RFC 6265
// matching rules and can be listed and edited, unlike net/http/cookiejar.
package cookies

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"

	"github.com/chapar-rest/chapar/internal/domain"
)

type key struct{ domain, path, name string }

func keyOf(c *domain.Cookie) key { return key{c.Domain, c.Path, c.Name} }

// Jar implements http.CookieJar and exposes its contents for editing.
type Jar struct {
	mu      sync.Mutex
	cookies map[key]*domain.Cookie
	now     func() time.Time
	dirty   bool
	version uint64
}

func NewJar(initial []*domain.Cookie) *Jar {
	j := &Jar{cookies: map[key]*domain.Cookie{}, now: time.Now}
	for _, c := range initial {
		if c == nil || c.Name == "" {
			continue
		}
		j.cookies[keyOf(c)] = c
	}
	return j
}

// SetCookies implements http.CookieJar.
func (j *Jar) SetCookies(u *url.URL, cs []*http.Cookie) {
	for _, c := range cs {
		j.set(u, c)
	}
}

// Cookies implements http.CookieJar.
func (j *Jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()

	now := j.now()
	host := canonicalHost(u.Hostname())
	secure := u.Scheme == "https" || u.Scheme == "wss" || isLocal(host)
	reqPath := u.EscapedPath()
	if reqPath == "" {
		reqPath = "/"
	}

	var matched []*domain.Cookie
	for k, c := range j.cookies {
		if c.Expired(now) {
			delete(j.cookies, k)
			j.changed()
			continue
		}
		if !c.Enabled || (c.Secure && !secure) {
			continue
		}
		if !domainMatch(c, host) || !pathMatch(reqPath, c.Path) {
			continue
		}
		// Access time alone does not mark the jar dirty; it is saved with the next change.
		c.LastAccessed = now
		matched = append(matched, c)
	}

	// RFC 6265 5.4: longer paths first, then older cookies first.
	sort.Slice(matched, func(a, b int) bool {
		if len(matched[a].Path) != len(matched[b].Path) {
			return len(matched[a].Path) > len(matched[b].Path)
		}
		return matched[a].Created.Before(matched[b].Created)
	})

	out := make([]*http.Cookie, len(matched))
	for i, c := range matched {
		out[i] = &http.Cookie{Name: c.Name, Value: c.Value}
	}
	return out
}

type Outcome string

const (
	OutcomeNew      Outcome = "new"
	OutcomeUpdated  Outcome = "updated"
	OutcomeDeleted  Outcome = "deleted"
	OutcomeRejected Outcome = "rejected"
)

// Event describes what the jar did with one Set-Cookie header.
type Event struct {
	URL     string
	Cookie  domain.Cookie
	Outcome Outcome
	Reason  string // set when rejected
}

func (j *Jar) set(u *url.URL, hc *http.Cookie) Event {
	j.mu.Lock()
	defer j.mu.Unlock()

	now := j.now()
	host := canonicalHost(u.Hostname())
	c := &domain.Cookie{
		Name:     hc.Name,
		Value:    hc.Value,
		Path:     hc.Path,
		Secure:   hc.Secure,
		HttpOnly: hc.HttpOnly,
		SameSite: sameSiteString(hc.SameSite),
		Enabled:  true,
		Source:   domain.CookieSourceResponse,
		Created:  now,
	}
	ev := Event{URL: u.String(), Outcome: OutcomeRejected}
	reject := func(reason string) Event {
		ev.Cookie, ev.Reason = *c, reason
		return ev
	}

	if hc.Domain == "" {
		c.Domain, c.HostOnly = host, true
	} else {
		d := canonicalHost(strings.TrimPrefix(hc.Domain, "."))
		if d != host && isPublicSuffix(d) {
			c.Domain = d
			return reject("domain " + d + " is a public suffix")
		}
		if !hasDomainSuffix(host, d) {
			c.Domain = d
			return reject("domain " + d + " does not match host " + host)
		}
		c.Domain = d
	}

	if c.Path == "" || c.Path[0] != '/' {
		c.Path = defaultPath(u.EscapedPath())
	}

	if c.Secure && u.Scheme != "https" && u.Scheme != "wss" && !isLocal(host) {
		return reject("Secure cookie set over an insecure connection")
	}

	// Max-Age wins over Expires.
	switch {
	case hc.MaxAge < 0:
		c.Expires = time.Unix(1, 0)
	case hc.MaxAge > 0:
		c.Expires = now.Add(time.Duration(hc.MaxAge) * time.Second)
	case !hc.Expires.IsZero():
		c.Expires = hc.Expires
	}

	k := keyOf(c)
	old := j.cookies[k]

	if c.Expired(now) {
		ev.Cookie = *c
		if old == nil {
			ev.Reason = "already expired"
			return ev
		}
		delete(j.cookies, k)
		j.changed()
		ev.Outcome = OutcomeDeleted
		return ev
	}

	if old != nil {
		c.ID, c.Created, c.Enabled = old.ID, old.Created, old.Enabled
		ev.Outcome = OutcomeUpdated
	} else {
		c.ID = uuid.NewString()
		ev.Outcome = OutcomeNew
	}
	c.LastAccessed = now
	j.cookies[k] = c
	j.changed()
	ev.Cookie = *c
	return ev
}

// All returns a copy of every cookie, sorted by domain, path and name.
func (j *Jar) All() []domain.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]domain.Cookie, 0, len(j.cookies))
	for _, c := range j.cookies {
		out = append(out, *c)
	}
	sort.Slice(out, func(a, b int) bool {
		x, y := out[a], out[b]
		if x.Domain != y.Domain {
			return x.Domain < y.Domain
		}
		if x.Path != y.Path {
			return x.Path < y.Path
		}
		return x.Name < y.Name
	})
	return out
}

var ErrInvalidCookie = errors.New("cookie needs a name and a domain")

// Upsert adds a cookie or replaces the one with the same ID. It also replaces
// a different cookie that has the same name, domain and path.
func (j *Jar) Upsert(c domain.Cookie) (domain.Cookie, error) {
	c.Name = strings.TrimSpace(c.Name)
	c.Domain = canonicalHost(strings.TrimPrefix(strings.TrimSpace(c.Domain), "."))
	if c.Name == "" || c.Domain == "" {
		return c, ErrInvalidCookie
	}
	if c.Path == "" || c.Path[0] != '/' {
		c.Path = "/"
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	now := j.now()
	if c.ID == "" {
		c.ID = uuid.NewString()
		c.Created = now
		if c.Source == "" {
			c.Source = domain.CookieSourceManual
		}
	}
	if c.Created.IsZero() {
		c.Created = now
	}
	for k, old := range j.cookies {
		if old.ID == c.ID {
			delete(j.cookies, k)
		}
	}
	j.cookies[keyOf(&c)] = &c
	j.changed()
	return c, nil
}

func (j *Jar) Delete(ids ...string) {
	j.removeWhere(func(c *domain.Cookie) bool { return slices.Contains(ids, c.ID) })
}

func (j *Jar) ClearDomain(d string) {
	d = canonicalHost(strings.TrimPrefix(d, "."))
	j.removeWhere(func(c *domain.Cookie) bool { return c.Domain == d })
}

func (j *Jar) ClearAll() { j.removeWhere(func(*domain.Cookie) bool { return true }) }

func (j *Jar) ClearExpired() {
	now := j.now()
	j.removeWhere(func(c *domain.Cookie) bool { return c.Expired(now) })
}

func (j *Jar) ClearSession() {
	j.removeWhere(func(c *domain.Cookie) bool { return c.Expires.IsZero() })
}

func (j *Jar) removeWhere(fn func(*domain.Cookie) bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for k, c := range j.cookies {
		if fn(c) {
			delete(j.cookies, k)
			j.changed()
		}
	}
}

// snapshot returns the cookies to persist and clears the dirty flag.
func (j *Jar) snapshot() ([]*domain.Cookie, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.dirty {
		return nil, false
	}
	list := make([]*domain.Cookie, 0, len(j.cookies))
	for _, c := range j.cookies {
		cp := *c
		list = append(list, &cp)
	}
	sort.Slice(list, func(a, b int) bool { return list[a].ID < list[b].ID })
	j.dirty = false
	return list, true
}

func (j *Jar) markDirty() {
	j.mu.Lock()
	j.dirty = true
	j.mu.Unlock()
}

// changed records a change to the jar. Callers hold j.mu.
func (j *Jar) changed() {
	j.dirty = true
	j.version++
}

// Version increases whenever cookies are added, changed or removed, so views
// can tell when to reload.
func (j *Jar) Version() uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.version
}

func canonicalHost(h string) string { return strings.ToLower(strings.TrimSuffix(h, ".")) }

// isLocal reports whether host is a loopback address, which browsers treat as
// a secure context even over plain http.
func isLocal(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isPublicSuffix(d string) bool {
	// IPs and single labels such as localhost or intranet hosts are allowed.
	if net.ParseIP(d) != nil || !strings.Contains(d, ".") {
		return false
	}
	ps, _ := publicsuffix.PublicSuffix(d)
	return ps == d
}

func hasDomainSuffix(host, d string) bool {
	if host == d {
		return true
	}
	if net.ParseIP(host) != nil {
		return false
	}
	return strings.HasSuffix(host, "."+d)
}

func domainMatch(c *domain.Cookie, host string) bool {
	if c.HostOnly {
		return host == c.Domain
	}
	return hasDomainSuffix(host, c.Domain)
}

func pathMatch(reqPath, cookiePath string) bool {
	if reqPath == cookiePath {
		return true
	}
	if !strings.HasPrefix(reqPath, cookiePath) {
		return false
	}
	return strings.HasSuffix(cookiePath, "/") || reqPath[len(cookiePath)] == '/'
}

func defaultPath(p string) string {
	if p == "" || p[0] != '/' {
		return "/"
	}
	i := strings.LastIndex(p, "/")
	if i == 0 {
		return "/"
	}
	return path.Clean(p[:i])
}

func sameSiteString(s http.SameSite) string {
	switch s {
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	}
	return ""
}
