package container

import (
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

// An environment opened for editing keeps its value IDs, so saving it
// rewrites only what the user changed.
func TestCopyEnvKeepsValueIDs(t *testing.T) {
	env := domain.NewEnvironment("dev")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Enable: true, Secret: true, Unset: true},
	}
	cp := CopyEnv(env)
	if cp.Spec.Values[0].ID != "1" || cp.Spec.Values[1].ID != "2" {
		t.Fatalf("IDs changed: %+v", cp.Spec.Values)
	}
	if !cp.Spec.Values[1].Unset {
		t.Fatal("the copy lost the unset mark")
	}
	cp.Spec.Values[0].Value = "changed"
	if env.Spec.Values[0].Value != "example.com" {
		t.Fatal("editing the copy changed the original")
	}
}
