package domain

import "github.com/google/uuid"

type Collection struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       ColSpec  `yaml:"spec"`

	FilePath string `yaml:"-"`
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
			Name: c.MetaData.Name,
		},
		Spec: ColSpec{
			Requests: make([]*Request, len(c.Spec.Requests)),
		},
		FilePath: c.FilePath,
	}

	for i, v := range c.Spec.Requests {
		clone.Spec.Requests[i] = v
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
		FilePath: "",
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
