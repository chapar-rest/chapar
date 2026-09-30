package testrun

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/secret"
)

// BundleOptions says what goes into a bundle besides the test cases.
type BundleOptions struct {
	// Env is exported with the cases when set.
	Env *domain.Environment
	// IncludeSecrets exports secret values in plain text. Off, they are
	// left out and named in SecretsLeftOut.
	IncludeSecrets bool
}

// Bundle packs cases with the requests they send, including the ones
// their pre-request actions trigger, so they run without the workspace.
// collection finds the collection a request belongs to.
func Bundle(name string, cases []*domain.TestCase, src RequestSource, collection func(id string) *domain.Collection, o BundleOptions) (*domain.TestBundle, error) {
	b := &domain.TestBundle{
		ApiVersion: domain.ApiVersion,
		Kind:       domain.KindTestBundle,
		MetaData:   domain.MetaData{Name: name},
	}

	seen := map[string]bool{}
	cols := map[string]*domain.Collection{}
	var add func(req *domain.Request) error
	add = func(req *domain.Request) error {
		if seen[req.ID()] {
			return nil
		}
		seen[req.ID()] = true
		cp := req.Clone()
		cp.MetaData.ID = req.ID()
		if req.CollectionID == "" {
			b.Spec.Requests = append(b.Spec.Requests, cp)
		} else {
			col := cols[req.CollectionID]
			if col == nil {
				orig := collection(req.CollectionID)
				if orig == nil {
					return fmt.Errorf("collection of request %s not found", RefOf(req))
				}
				col = &domain.Collection{
					ApiVersion: orig.ApiVersion,
					Kind:       orig.Kind,
					MetaData:   orig.MetaData,
					Spec:       domain.ColSpec{Headers: orig.Spec.Headers, Auth: orig.Spec.Auth},
				}
				cols[req.CollectionID] = col
				b.Spec.Collections = append(b.Spec.Collections, col)
			}
			col.Spec.Requests = append(col.Spec.Requests, cp)
		}
		// A pre-request action may send another request first.
		if pre := req.Spec.GetPreRequest(); pre.TriggerRequest != nil && pre.TriggerRequest.RequestID != "" {
			trig := src.RequestByID(pre.TriggerRequest.RequestID)
			if trig == nil {
				return fmt.Errorf("request %s triggers request %s, which was not found", RefOf(req), pre.TriggerRequest.RequestID)
			}
			return add(trig)
		}
		return nil
	}

	for _, tc := range cases {
		for _, s := range tc.AllSteps() {
			req, err := Resolve(src, s.Request)
			if err != nil {
				return nil, fmt.Errorf("%s: step %s: %w", tc.GetName(), s.ID, err)
			}
			if err := add(req); err != nil {
				return nil, fmt.Errorf("%s: %w", tc.GetName(), err)
			}
		}
		cp := tc.Clone()
		cp.MetaData.ID = tc.ID()
		b.Spec.TestCases = append(b.Spec.TestCases, cp)
	}

	if o.Env != nil {
		env := &domain.Environment{ApiVersion: o.Env.ApiVersion, Kind: o.Env.Kind, MetaData: o.Env.MetaData}
		for _, kv := range o.Env.Spec.Values {
			secretValue := kv.Secret || kv.Locked || secret.IsEncrypted(kv.Value)
			if secretValue && (!o.IncludeSecrets || kv.Locked) {
				b.Spec.SecretsLeftOut = append(b.Spec.SecretsLeftOut, kv.Key)
				continue
			}
			kv.Secret, kv.Locked = false, false
			env.Spec.Values = append(env.Spec.Values, kv)
		}
		b.Spec.Environment = env
	}
	return b, nil
}
