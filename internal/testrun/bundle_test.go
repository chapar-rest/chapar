package testrun

import (
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/secret"
)

func TestBundle(t *testing.T) {
	col := domain.NewCollection("Todos")
	col.MetaData.ID = "col"
	col.Spec.Headers = []domain.KeyValue{{Key: "X-Team", Value: "core", Enable: true}}

	login := httpReq("login", "", "Login", "POST", "{{base}}/login", "")
	create := httpReq("create", "Todos", "Create", "POST", "{{base}}/todos", "")
	create.CollectionID = "col"
	create.Spec.HTTP.Request.PreRequest = domain.PreRequest{
		Type:           domain.PrePostTypeTriggerRequest,
		TriggerRequest: &domain.TriggerRequest{RequestID: "login"},
	}
	get := httpReq("get", "Todos", "Get", "GET", "{{base}}/todos/1", "")
	get.CollectionID = "col"
	unused := httpReq("unused", "", "Unused", "GET", "{{base}}/x", "")
	src := requests{login, create, get, unused}
	cols := func(id string) *domain.Collection {
		if id == "col" {
			return col
		}
		return nil
	}

	tc := domain.NewTestCase("todos")
	tc.Spec.Steps = []domain.TestStep{
		{ID: "a", Request: domain.TestRequestRef{Ref: "Todos/Create"}},
		{ID: "b", Request: domain.TestRequestRef{ID: "get"}},
	}
	tc.Spec.Teardown = []domain.TestStep{{ID: "c", Request: domain.TestRequestRef{ID: "create"}}}

	env := domain.NewEnvironment("dev")
	env.Spec.Values = []domain.KeyValue{
		{Key: "base", Value: "http://x", Enable: true},
		{Key: "token", Value: "s3cret", Enable: true, Secret: true},
		{Key: "locked", Value: secret.Prefix + "AAAA", Enable: true, Locked: true},
	}

	b, err := Bundle("todos", []*domain.TestCase{tc}, src, cols, BundleOptions{Env: env})
	if err != nil {
		t.Fatal(err)
	}
	if b.Kind != domain.KindTestBundle || len(b.Spec.TestCases) != 1 || b.Spec.TestCases[0].ID() != tc.ID() {
		t.Fatalf("bundle = %+v", b)
	}
	// login comes in because create triggers it; unused stays out.
	if len(b.Spec.Requests) != 1 || b.Spec.Requests[0].ID() != "login" {
		t.Errorf("standalone requests = %v", b.Spec.Requests)
	}
	if len(b.Spec.Collections) != 1 {
		t.Fatalf("collections = %v", b.Spec.Collections)
	}
	c := b.Spec.Collections[0]
	if c.ID() != "col" || len(c.Spec.Requests) != 2 || len(c.Spec.Headers) != 1 {
		t.Errorf("collection = %+v", c)
	}
	if got := strings.Join(b.Spec.SecretsLeftOut, ","); got != "token,locked" {
		t.Errorf("secrets left out = %q", got)
	}
	if len(b.Spec.Environment.Spec.Values) != 1 || len(env.Spec.Values) != 3 {
		t.Errorf("env = %+v", b.Spec.Environment.Spec.Values)
	}

	b, err = Bundle("todos", []*domain.TestCase{tc}, src, cols, BundleOptions{Env: env, IncludeSecrets: true})
	if err != nil {
		t.Fatal(err)
	}
	// A locked value can't be read, so it stays out even then.
	if got := strings.Join(b.Spec.SecretsLeftOut, ","); got != "locked" {
		t.Errorf("secrets left out = %q", got)
	}
	for _, kv := range b.Spec.Environment.Spec.Values {
		if kv.Key == "token" && (kv.Value != "s3cret" || kv.Secret) {
			t.Errorf("token = %+v", kv)
		}
	}

	tc.Spec.Steps = append(tc.Spec.Steps, domain.TestStep{ID: "d", Request: domain.TestRequestRef{Ref: "Nope"}})
	if _, err := Bundle("todos", []*domain.TestCase{tc}, src, cols, BundleOptions{}); err == nil || !strings.Contains(err.Error(), `step d: request "Nope" not found`) {
		t.Errorf("missing request: err = %v", err)
	}
}
