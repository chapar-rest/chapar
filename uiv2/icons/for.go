package icons

import (
	"github.com/chapar-rest/chapar/internal/domain"
	yogaicons "github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

const (
	nameGET  = "req-get"
	namePOST = "req-post"
	namePUT  = "req-put"
	nameDEL  = "req-del"
	namePAT  = "req-pat"
	nameHEAD = "req-head"
	nameOPT  = "req-opt"
	nameCON  = "req-con"
	nameTRC  = "req-trc"
	nameGRPC = "req-grpc"
	nameGQL  = "req-gql"
	nameUnk  = "req-unk"
)

var (
	patchColor   = render.RGBA8(0x9c, 0x27, 0xb0, 0xff)
	optionsColor = render.RGBA8(0x00, 0x80, 0x80, 0xff)
)

func icon(name string) yogaicons.Icon {
	return yogaicons.Icon{Name: name}
}

// Badge returns the atlas icon name for a request tree row.
func Badge(req *domain.Request) yogaicons.Icon {
	if req == nil {
		return icon(nameUnk)
	}
	switch req.MetaData.Type {
	case domain.RequestTypeGRPC:
		return icon(nameGRPC)
	case domain.RequestTypeGraphQL:
		return icon(nameGQL)
	case domain.RequestTypeHTTP:
		if req.Spec.HTTP == nil {
			return icon(nameGET)
		}
		switch req.Spec.HTTP.Method {
		case domain.RequestMethodGET:
			return icon(nameGET)
		case domain.RequestMethodPOST:
			return icon(namePOST)
		case domain.RequestMethodPUT:
			return icon(namePUT)
		case domain.RequestMethodDELETE:
			return icon(nameDEL)
		case domain.RequestMethodPATCH:
			return icon(namePAT)
		case domain.RequestMethodHEAD:
			return icon(nameHEAD)
		case domain.RequestMethodOPTIONS:
			return icon(nameOPT)
		case domain.RequestMethodCONNECT:
			return icon(nameCON)
		case domain.RequestMethodTRACE:
			return icon(nameTRC)
		default:
			return icon(nameUnk)
		}
	default:
		return icon(nameUnk)
	}
}

// Color returns the tint for a request badge, using live theme tokens.
func Color(req *domain.Request, th *theme.Theme) render.Color {
	if req == nil || th == nil {
		return render.RGBA8(0x80, 0x80, 0x80, 0xff)
	}
	switch req.MetaData.Type {
	case domain.RequestTypeGRPC:
		return th.Success
	case domain.RequestTypeGraphQL:
		return th.Accent
	case domain.RequestTypeHTTP:
		if req.Spec.HTTP == nil {
			return th.Success
		}
		switch req.Spec.HTTP.Method {
		case domain.RequestMethodGET:
			return th.Success
		case domain.RequestMethodPOST:
			return th.Accent
		case domain.RequestMethodPUT:
			return th.Warning
		case domain.RequestMethodDELETE:
			return th.Error
		case domain.RequestMethodPATCH:
			return patchColor
		case domain.RequestMethodOPTIONS:
			return optionsColor
		case domain.RequestMethodHEAD, domain.RequestMethodCONNECT, domain.RequestMethodTRACE:
			return th.ForegroundMuted
		default:
			return th.ForegroundMuted
		}
	default:
		return th.ForegroundMuted
	}
}

// For returns badge icon and tint for a request.
func For(req *domain.Request) (yogaicons.Icon, render.Color) {
	th := theme.Current()
	return Badge(req), Color(req, th)
}
