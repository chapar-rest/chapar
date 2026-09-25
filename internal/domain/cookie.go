package domain

import "time"

type CookieSource string

const (
	CookieSourceManual   CookieSource = "manual"
	CookieSourceResponse CookieSource = "response"
	CookieSourceImported CookieSource = "imported"
)

// Cookie is a cookie held in an environment's cookie jar.
type Cookie struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Value    string       `json:"value"`
	Domain   string       `json:"domain"` // lowercase, no leading dot
	HostOnly bool         `json:"hostOnly"`
	Path     string       `json:"path"`
	Expires  time.Time    `json:"expires,omitzero"` // zero means a session cookie
	Secure   bool         `json:"secure"`
	HttpOnly bool         `json:"httpOnly"`
	SameSite string       `json:"sameSite,omitempty"` // "", Lax, Strict or None
	Enabled  bool         `json:"enabled"`            // disabled cookies are kept but not sent
	Source   CookieSource `json:"source"`

	Created      time.Time `json:"created"`
	LastAccessed time.Time `json:"lastAccessed"`
}

func (c *Cookie) Expired(now time.Time) bool {
	return !c.Expires.IsZero() && !c.Expires.After(now)
}
