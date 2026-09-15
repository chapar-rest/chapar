package container

import "github.com/chapar-rest/chapar/internal/egress"

// DisplayBody returns the bytes to show in a response editor. Pretty-printed
// JSON is preferred when egress already produced it; otherwise the raw body.
func DisplayBody(res *egress.Response) []byte {
	if res == nil {
		return nil
	}
	if res.IsJSON && res.JSON != "" {
		return []byte(res.JSON)
	}
	if len(res.Body) > 0 {
		return res.Body
	}
	return []byte(res.JSON)
}
