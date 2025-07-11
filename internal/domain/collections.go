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

func (c *Collection) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(c)
}

type ColSpec struct {
	Requests []*Request `yaml:"requests"`
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
