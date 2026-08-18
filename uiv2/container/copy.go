package container

import "github.com/chapar-rest/chapar/internal/domain"

func CopyRequest(r *domain.Request) *domain.Request {
	if r == nil {
		return nil
	}
	out := *r
	spec := r.Spec
	if r.Spec.HTTP != nil {
		spec.HTTP = r.Spec.HTTP.Clone()
		if spec.HTTP.Request == nil {
			spec.HTTP.Request = &domain.HTTPRequest{}
		}
	}
	if r.Spec.GRPC != nil {
		spec.GRPC = r.Spec.GRPC.Clone()
	}
	if r.Spec.GraphQL != nil {
		spec.GraphQL = r.Spec.GraphQL.Clone()
	}
	out.Spec = spec
	return &out
}

func CopyEnv(e *domain.Environment) *domain.Environment {
	if e == nil {
		return nil
	}
	return &domain.Environment{
		ApiVersion: e.ApiVersion,
		Kind:       e.Kind,
		MetaData:   e.MetaData,
		Spec:       e.Spec.Clone(),
	}
}

func CopyCollection(c *domain.Collection) *domain.Collection {
	if c == nil {
		return nil
	}
	out := *c
	if c.Spec.Headers != nil {
		out.Spec.Headers = append([]domain.KeyValue(nil), c.Spec.Headers...)
	}
	if c.Spec.Requests != nil {
		out.Spec.Requests = append([]*domain.Request(nil), c.Spec.Requests...)
	}
	out.Spec.Auth = c.Spec.Auth.Clone()
	out.Spec.Notes = c.Spec.Notes
	return &out
}
