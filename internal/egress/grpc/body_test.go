package grpc

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestBlankBodyIsEmptyMessage(t *testing.T) {
	for _, body := range []string{"", "  \n", "{}"} {
		if err := protojson.Unmarshal(messageJSON(body), &emptypb.Empty{}); err != nil {
			t.Errorf("body %q: %v", body, err)
		}
	}
}
