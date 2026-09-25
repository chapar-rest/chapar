package rest

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
)

// SendObject sends an HTTP request from domain objects instead of looking them up in state.
func (s *Service) SendObject(req *domain.Request, env *domain.Environment, collection *domain.Collection) (*egress.Response, error) {
	if req == nil || req.Spec.HTTP == nil {
		return nil, fmt.Errorf("invalid http request")
	}

	r := req.Clone()
	r.MetaData.ID = req.MetaData.ID
	r.CollectionID = req.CollectionID
	r.CollectionName = req.CollectionName

	if collection != nil && r.Spec.HTTP != nil && r.Spec.HTTP.Request != nil {
		r.Spec.HTTP.Request.Headers = domain.MergeHeaders(collection.Spec.Headers, r.Spec.HTTP.Request.Headers)
		if r.Spec.HTTP.Request.Auth.Type == domain.AuthTypeInherit {
			r.Spec.HTTP.Request.Auth = collection.Spec.Auth
		}
	}

	return s.sendRequest(r.Spec.HTTP, env)
}
