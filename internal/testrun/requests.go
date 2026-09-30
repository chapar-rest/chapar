package testrun

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
)

// RequestSource is where a runner finds the requests steps send.
type RequestSource interface {
	RequestByID(id string) *domain.Request
	// AllRequests returns standalone requests and those in collections,
	// with CollectionName set on the latter.
	AllRequests() []*domain.Request
}

// RefOf returns the ref that names req: "Collection/Request", or the
// request name for one outside a collection.
func RefOf(req *domain.Request) string {
	if req.CollectionName != "" {
		return req.CollectionName + "/" + req.MetaData.Name
	}
	return req.MetaData.Name
}

// Resolve finds the request ref points at: by ID first, then by name.
func Resolve(src RequestSource, ref domain.TestRequestRef) (*domain.Request, error) {
	if ref.ID != "" {
		if req := src.RequestByID(ref.ID); req != nil {
			return req, nil
		}
		if ref.Ref == "" {
			return nil, fmt.Errorf("request %s not found", ref.ID)
		}
	}

	var found *domain.Request
	n := 0
	for _, req := range src.AllRequests() {
		if RefOf(req) == ref.Ref {
			found = req
			n++
		}
	}
	switch n {
	case 0:
		return nil, fmt.Errorf("request %q not found", ref.Ref)
	case 1:
		return found, nil
	}
	return nil, fmt.Errorf("%d requests are named %q; set request.id", n, ref.Ref)
}

// Finder checks step requests against src, for Validate.
func Finder(src RequestSource) RequestFinder {
	return func(ref domain.TestRequestRef) error {
		_, err := Resolve(src, ref)
		return err
	}
}
