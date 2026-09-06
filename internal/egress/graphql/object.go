package graphql

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
)

// SendObject sends a GraphQL request from domain objects instead of looking them up in state.
func (s *Service) SendObject(req *domain.Request, env *domain.Environment, collection *domain.Collection) (*egress.Response, error) {
	if req == nil || req.Spec.GraphQL == nil {
		return nil, fmt.Errorf("invalid graphql request")
	}

	r := req.Clone()
	r.MetaData.ID = req.MetaData.ID
	r.CollectionID = req.CollectionID
	r.CollectionName = req.CollectionName

	if collection != nil && r.Spec.GraphQL != nil {
		r.Spec.GraphQL.Headers = domain.MergeHeaders(collection.Spec.Headers, r.Spec.GraphQL.Headers)
		if r.Spec.GraphQL.Auth.Type == domain.AuthTypeInherit {
			r.Spec.GraphQL.Auth = collection.Spec.Auth
		}
	}

	return s.sendRequest(r.Spec.GraphQL, env)
}
