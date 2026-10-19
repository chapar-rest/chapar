package domain

import (
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type Collection struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       ColSpec  `yaml:"spec"`
}

func (c *Collection) ID() string {
	return c.MetaData.ID
}

func (c *Collection) GetKind() string {
	return c.Kind
}

func (c *Collection) SetName(name string) {
	c.MetaData.Name = name
}

func (c *Collection) GetName() string {
	return c.MetaData.Name
}

// MarshalYaml writes the collection's own file. Its requests are kept in
// files of their own next to it, so they are left out here: a copy in this
// file would go stale and show up in every diff.
func (c *Collection) MarshalYaml() ([]byte, error) {
	out := *c
	out.Spec.Requests = []*Request{}
	return yaml.Marshal(&out)
}

type ColSpec struct {
	Requests []*Request `yaml:"requests"`
	Headers  []KeyValue `yaml:"headers"`
	Auth     Auth       `yaml:"auth"`
	Notes    string     `yaml:"notes"`
}

func (c *Collection) Clone() *Collection {
	clone := &Collection{
		ApiVersion: c.ApiVersion,
		Kind:       c.Kind,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: c.MetaData.Name + "(copy)",
		},
		Spec: ColSpec{
			Requests: make([]*Request, 0, len(c.Spec.Requests)),
		},
	}

	// Clone headers
	if len(c.Spec.Headers) > 0 {
		clone.Spec.Headers = make([]KeyValue, len(c.Spec.Headers))
		copy(clone.Spec.Headers, c.Spec.Headers)
	}

	// Clone auth
	if c.Spec.Auth != (Auth{}) {
		clone.Spec.Auth = c.Spec.Auth.Clone()
	}

	// Clone notes
	clone.Spec.Notes = c.Spec.Notes

	for _, req := range c.Spec.Requests {
		cloneReq := req.Clone()
		cloneReq.CollectionID = clone.MetaData.ID
		cloneReq.CollectionName = clone.MetaData.Name
		clone.Spec.Requests = append(clone.Spec.Requests, cloneReq)
	}

	return clone
}

func NewCollection(name string) *Collection {
	return &Collection{
		ApiVersion: ApiVersion,
		Kind:       KindCollection,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
		Spec: ColSpec{
			Requests: make([]*Request, 0),
		},
	}
}

func (c *Collection) AddRequest(req *Request) {
	c.Spec.Requests = append(c.Spec.Requests, req)
}

func (c *Collection) RemoveRequest(req *Request) {
	for i, r := range c.Spec.Requests {
		if r.MetaData.ID == req.MetaData.ID {
			c.Spec.Requests = append(c.Spec.Requests[:i], c.Spec.Requests[i+1:]...)
			return
		}
	}
}

func (c *Collection) FindRequestByID(id string) *Request {
	for _, r := range c.Spec.Requests {
		if r.MetaData.ID == id {
			return r
		}
	}
	return nil
}
