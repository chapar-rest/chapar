package egress

import (
	"net/http"
	"sort"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/scripting"
)

// ScriptData is what a post-request script sees of the response.
func (r *Response) ScriptData() *scripting.ResponseData {
	if r == nil {
		return nil
	}
	code := r.StatusCode
	if code == 0 {
		code = r.StatueCode
	}
	size := r.Size
	if size == 0 {
		size = len(r.Body)
	}
	out := &scripting.ResponseData{
		StatusCode: code,
		Status:     r.Status,
		Headers:    mapPairs(r.ResponseHeaders),
		Body:       string(r.Body),
		ElapsedMS:  float64(r.TimePassed) / float64(time.Millisecond),
		Size:       size,
		Metadata:   kvPairs(r.ResponseMetadata),
		Trailers:   kvPairs(r.Trailers),
	}
	if out.Status == "" && code > 0 && len(r.ResponseMetadata) == 0 {
		out.Status = http.StatusText(code)
	}
	if r.Error != nil {
		out.Error = r.Error.Error()
	}
	for _, c := range r.Cookies {
		sc := scripting.Cookie{
			Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
			Secure: c.Secure, HTTPOnly: c.HttpOnly,
		}
		if !c.Expires.IsZero() {
			sc.Expires = c.Expires.UTC().Format(time.RFC3339)
		}
		out.Cookies = append(out.Cookies, sc)
	}
	return out
}

func mapPairs(m map[string]string) []scripting.Pair {
	out := make([]scripting.Pair, 0, len(m))
	for k, v := range m {
		out = append(out, scripting.Pair{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

func kvPairs(kvs []domain.KeyValue) []scripting.Pair {
	if len(kvs) == 0 {
		return nil
	}
	out := make([]scripting.Pair, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, scripting.Pair{kv.Key, kv.Value})
	}
	return out
}
