package container

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// CookiesState holds the response Cookies tab: what the server set and what
// the cookie jar sent.
type CookiesState struct {
	received *ui.Table
	sent     *ui.Table
	raw      *ui.Editor

	have   bool // a response was received
	hasJar bool // the response went through a cookie jar

	clip func() input.Clipboard
}

func (s *CookiesState) ensure(deps Deps) {
	if s.received != nil {
		return
	}
	copyValue := func(t *ui.Table) func(string) {
		return func(rowID string) {
			for _, r := range t.Rows {
				if r.ID == rowID {
					s.copy(deps, r.Cells["value"])
					return
				}
			}
		}
	}

	s.received = ui.NewTable([]ui.TableColumn{
		{ID: "status", Label: "Status", Kind: ui.TableColText, Width: 80, Sortable: true},
		{ID: "name", Label: "Name", Kind: ui.TableColText, Width: 140, Sortable: true},
		{ID: "value", Label: "Value", Kind: ui.TableColText},
		{ID: "domain", Label: "Domain", Kind: ui.TableColText, Width: 140, Sortable: true},
		{ID: "path", Label: "Path", Kind: ui.TableColText, Width: 70},
		{ID: "expires", Label: "Expires", Kind: ui.TableColText, Width: 110},
		{ID: "note", Label: "Flags / reason", Kind: ui.TableColText},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Copy, Tooltip: "Copy value"}})
	s.received.Actions[0].OnClick = copyValue(s.received)
	s.received.Editable = false
	s.received.CollapseEmpty = true
	s.received.MinHeight = 120

	s.sent = ui.NewTable([]ui.TableColumn{
		{ID: "name", Label: "Name", Kind: ui.TableColText, Width: 180},
		{ID: "value", Label: "Value", Kind: ui.TableColText},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Copy, Tooltip: "Copy value"}})
	s.sent.Actions[0].OnClick = copyValue(s.sent)
	s.sent.Editable = false
	s.sent.CollapseEmpty = true
	s.sent.MinHeight = 80

	s.raw = ui.NewEditor(nil, highlight.Noop{}, ui.WithSoftWrap(true))
}

// Set fills the tab from a response. A nil response clears it.
func (s *CookiesState) Set(deps Deps, res *egress.Response) {
	s.ensure(deps)
	s.have = res != nil
	s.hasJar = false
	if res == nil {
		s.received.SetRows(nil)
		s.sent.SetRows(nil)
		s.raw = ReplaceEditor(s.raw, nil, highlight.Noop{})
		return
	}

	now := time.Now()
	var rows []ui.TableRow
	if len(res.CookieEvents) > 0 || len(res.SentCookies) > 0 {
		s.hasJar = true
	}
	if len(res.CookieEvents) > 0 {
		for i, ev := range res.CookieEvents {
			c := ev.Cookie
			note := CookieFlags(c)
			if ev.Reason != "" {
				note = ev.Reason
			}
			rows = append(rows, ui.TableRow{
				ID: fmt.Sprintf("ev-%d", i),
				Cells: map[string]string{
					"status":  string(ev.Outcome),
					"name":    c.Name,
					"value":   c.Value,
					"domain":  c.Domain,
					"path":    c.Path,
					"expires": CookieExpiry(c, now),
					"note":    note,
				},
			})
		}
	} else {
		// No jar: show the final response's Set-Cookie headers as parsed.
		for i, hc := range res.Cookies {
			c := domain.Cookie{Name: hc.Name, Value: hc.Value, Domain: hc.Domain, Path: hc.Path,
				Secure: hc.Secure, HttpOnly: hc.HttpOnly, Enabled: true}
			switch {
			case hc.MaxAge < 0:
				c.Expires = time.Unix(1, 0)
			case hc.MaxAge > 0:
				c.Expires = now.Add(time.Duration(hc.MaxAge) * time.Second)
			default:
				c.Expires = hc.Expires
			}
			rows = append(rows, ui.TableRow{
				ID: fmt.Sprintf("ck-%d", i),
				Cells: map[string]string{
					"status":  "",
					"name":    c.Name,
					"value":   c.Value,
					"domain":  c.Domain,
					"path":    c.Path,
					"expires": CookieExpiry(c, now),
					"note":    CookieFlags(c),
				},
			})
		}
	}
	s.received.SetRows(rows)

	sent := make([]ui.TableRow, 0, len(res.SentCookies))
	for i, c := range res.SentCookies {
		sent = append(sent, ui.TableRow{
			ID:    fmt.Sprintf("sent-%d", i),
			Cells: map[string]string{"name": c.Name, "value": c.Value},
		})
	}
	s.sent.SetRows(sent)

	s.raw = ReplaceEditor(s.raw, []byte(rawCookies(res)), highlight.Noop{})
}

// Close releases the tab's widgets.
func (s *CookiesState) Close() {
	if s.raw != nil {
		s.raw.Close()
	}
}

// rawCookies lists the Set-Cookie headers of the final response and the
// Cookie header the jar produced.
func rawCookies(res *egress.Response) string {
	var b strings.Builder
	for _, c := range res.Cookies {
		line := c.Raw
		if line == "" {
			line = c.String()
		}
		fmt.Fprintf(&b, "Set-Cookie: %s\n", line)
	}
	if len(res.SentCookies) > 0 {
		parts := make([]string, len(res.SentCookies))
		for i, c := range res.SentCookies {
			parts[i] = (&http.Cookie{Name: c.Name, Value: c.Value}).String()
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "# Sent from cookie jar\nCookie: %s\n", strings.Join(parts, "; "))
	}
	return b.String()
}

// CookiesView is the response Cookies tab. raw shows the headers as text.
func CookiesView(id string, th *theme.Theme, ctx *ui.Ctx, deps Deps, s *CookiesState, raw bool) ui.View {
	s.ensure(deps)
	s.clip = ctx.Clipboard
	if raw {
		return ResponseEditorMenu(s.raw, ctx, deps, "cookies.txt")
	}

	var manage ui.View = ui.Spacer()
	if deps.ManageCookies != nil {
		manage = ui.Button(id+"-manage", ui.Text("Manage cookies")).
			IconStart(icons.Cookie).Subtle().OnClick(deps.ManageCookies)
	}

	if !s.have {
		return ui.Column(
			ui.EmptyState("No response yet", "Cookies the server sets show up here after you send the request.").
				EmptyIcon(icons.Cookie).Action(manage),
		).Grow(1)
	}

	received := ui.View(ui.ViewOf(s.received).Grow(1))
	if len(s.received.Rows) == 0 {
		received = ui.Caption("The server did not set any cookies.")
	}

	rows := []ui.View{
		ui.Row(ui.Strong(fmt.Sprintf("Received (%d)", len(s.received.Rows))), ui.Spacer(), manage).
			Align(ui.AlignCenter),
		received,
	}
	if s.hasJar {
		sent := ui.View(ui.ViewOf(s.sent))
		if len(s.sent.Rows) == 0 {
			sent = ui.Caption("No cookies from the jar matched this request.")
		}
		rows = append(rows,
			ui.Strong(fmt.Sprintf("Sent from jar (%d)", len(s.sent.Rows))).MarginTop(th.Spacing.S),
			sent,
		)
	}
	return ui.Column(rows...).Gap(th.Spacing.XS).Grow(1)
}

// CookieExpiry describes when a cookie expires relative to now.
func CookieExpiry(c domain.Cookie, now time.Time) string {
	if c.Expires.IsZero() {
		return "Session"
	}
	d := c.Expires.Sub(now)
	if d <= 0 {
		return "Expired"
	}
	return "in " + shortDuration(d)
}

func shortDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 2*365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/24/365))
	}
}

// CookieFlags lists a cookie's attributes compactly, e.g. "Secure HttpOnly Lax".
func CookieFlags(c domain.Cookie) string {
	var f []string
	if !c.Enabled {
		f = append(f, "Disabled")
	}
	if c.Secure {
		f = append(f, "Secure")
	}
	if c.HttpOnly {
		f = append(f, "HttpOnly")
	}
	if c.SameSite != "" {
		f = append(f, c.SameSite)
	}
	return strings.Join(f, " ")
}

func (s *CookiesState) copy(deps Deps, text string) {
	if s.clip == nil {
		return
	}
	if clip := s.clip(); clip != nil {
		clip.Set(text)
		deps.Toast("Copied")
	}
}
