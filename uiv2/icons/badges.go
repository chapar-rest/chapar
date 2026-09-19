package icons

import "fmt"

// badgeSVG builds a compact square badge rasterized as a single-color glyph.
func badgeSVG(label string) []byte {
	size := 11.5
	y := 17.0
	switch len(label) {
	case 4:
		size = 9.5
		y = 16.8
	case 5:
		size = 8.2
		y = 16.6
	default:
		size = 11.5
		y = 17.0
	}
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><text x="12" y="%.1f" text-anchor="middle" font-family="Arial,Helvetica,sans-serif" font-size="%.1f" font-weight="700" fill="currentColor">%s</text></svg>`, y, size, label))
}

var badgeLabels = map[string]string{
	"req-get":  "GET",
	"req-post": "POST",
	"req-put":  "PUT",
	"req-del":  "DEL",
	"req-pat":  "PAT",
	"req-head": "HEAD",
	"req-opt":  "OPT",
	"req-con":  "CON",
	"req-trc":  "TRC",
	"req-grpc": "gRPC",
	"req-gql":  "GQL",
	"req-unk":  "?",
}
