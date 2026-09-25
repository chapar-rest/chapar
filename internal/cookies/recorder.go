package cookies

import (
	"net/http"
	"net/url"
	"sync"
)

// Recorder is a per-request view of a Jar. It records what the jar sent and
// did with Set-Cookie headers, including on redirect hops, and leaves out jar
// cookies the request already sets itself.
type Recorder struct {
	jar  *Jar
	skip map[string]bool

	mu     sync.Mutex
	events []Event
	sent   []*http.Cookie
}

// Record returns a Recorder over j. Jar cookies named in explicit are not sent,
// so cookies set on the request win over jar cookies with the same name.
func (j *Jar) Record(explicit []string) *Recorder {
	r := &Recorder{jar: j, skip: map[string]bool{}}
	for _, n := range explicit {
		r.skip[n] = true
	}
	return r
}

func (r *Recorder) SetCookies(u *url.URL, cs []*http.Cookie) {
	for _, c := range cs {
		ev := r.jar.set(u, c)
		r.mu.Lock()
		r.events = append(r.events, ev)
		r.mu.Unlock()
	}
}

func (r *Recorder) Cookies(u *url.URL) []*http.Cookie {
	var out []*http.Cookie
	for _, c := range r.jar.Cookies(u) {
		if !r.skip[c.Name] {
			out = append(out, c)
		}
	}
	r.mu.Lock()
	r.sent = append(r.sent, out...)
	r.mu.Unlock()
	return out
}

// Events returns what happened to each Set-Cookie header received.
func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events...)
}

// Sent returns the jar cookies attached to outgoing requests.
func (r *Recorder) Sent() []*http.Cookie {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*http.Cookie(nil), r.sent...)
}

// Attach sets client.Jar to a recorder over the environment's jar, for
// sending req. Cookies req already carries in its Cookie header are not
// duplicated from the jar. A nil store attaches nothing and returns nil.
func (s *Store) Attach(envID string, client *http.Client, req *http.Request) (*Recorder, error) {
	if s == nil {
		return nil, nil
	}
	j, err := s.For(envID)
	if err != nil {
		return nil, err
	}
	var explicit []string
	for _, c := range req.Cookies() {
		explicit = append(explicit, c.Name)
	}
	r := j.Record(explicit)
	client.Jar = r
	return r, nil
}
