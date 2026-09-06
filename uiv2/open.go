package uiv2

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/container"
	colc "github.com/chapar-rest/chapar/uiv2/container/collection"
	envc "github.com/chapar-rest/chapar/uiv2/container/env"
	gqlc "github.com/chapar-rest/chapar/uiv2/container/graphql"
	grpcc "github.com/chapar-rest/chapar/uiv2/container/grpc"
	httpc "github.com/chapar-rest/chapar/uiv2/container/http"
)

func openContainer(spec container.OpenSpec) (container.Container, error) {
	switch {
	case spec.Env != nil:
		return envc.Open(spec.Env, spec.Deps), nil
	case spec.Collection != nil:
		return colc.Open(spec.Collection, spec.Deps), nil
	case spec.Request != nil:
		switch spec.Request.MetaData.Type {
		case domain.RequestTypeGRPC:
			return grpcc.Open(spec.Request, spec.Deps), nil
		case domain.RequestTypeGraphQL:
			return gqlc.Open(spec.Request, spec.Deps), nil
		default:
			return httpc.Open(spec.Request, spec.Deps), nil
		}
	default:
		return nil, fmt.Errorf("open spec has no document")
	}
}
