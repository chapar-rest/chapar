package icons

import (
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/render"
)

func TestBadgesRegistered(t *testing.T) {
	for name := range badgeLabels {
		if !render.HasSVGOverride(name) {
			t.Fatalf("badge %q not registered", name)
		}
	}
}

func TestBadgeHTTPMethods(t *testing.T) {
	cases := []struct {
		method string
		want   string
	}{
		{domain.RequestMethodGET, nameGET},
		{domain.RequestMethodPOST, namePOST},
		{domain.RequestMethodDELETE, nameDEL},
		{domain.RequestMethodPATCH, namePAT},
	}
	for _, tc := range cases {
		req := domain.NewHTTPRequest("x")
		req.Spec.HTTP.Method = tc.method
		got := Badge(req)
		if got.Name != tc.want {
			t.Fatalf("method %s: got %q want %q", tc.method, got.Name, tc.want)
		}
	}
}

func TestBadgeGRPC(t *testing.T) {
	req := domain.NewGRPCRequest("x")
	if got := Badge(req); got.Name != nameGRPC {
		t.Fatalf("got %q want %q", got.Name, nameGRPC)
	}
}

func TestBadgeSVGRasterizes(t *testing.T) {
	mask, err := render.RasterizeSVG(badgeSVG("GET"), 40)
	if err != nil {
		t.Fatal(err)
	}
	if mask == nil || len(mask.Pix) == 0 {
		t.Fatal("empty raster mask")
	}
}
