package rest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
)

func Test_applyVariables(t *testing.T) {
	sampleEnv := &domain.EnvSpec{
		Values: []domain.KeyValue{
			{
				ID:    "1",
				Key:   "key1",
				Value: "{{randomUUID4}}",
			},
		},
	}

	sampleReq := &domain.HTTPRequestSpec{}

	applyVariables(sampleReq, sampleEnv)

	if sampleEnv.Values[0].Value == "{{randomUUID4}}" {
		t.Errorf("expected randomUUID4 but got %s", sampleEnv.Values[0].Value)
	}

	_, err := uuid.Parse(sampleEnv.Values[0].Value)
	if err != nil {
		t.Errorf("expected valid uuid but got %s", sampleEnv.Values[0].Value)
	}
}
