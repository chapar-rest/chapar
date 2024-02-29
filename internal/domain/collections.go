package domain

type Collection struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       ColSpec  `yaml:"spec"`

	FilePath string `yaml:"-"`
}

type ColSpec struct {
	Requests []Request `yaml:"requests"`
}

func (c *Collection) Clone() *Collection {
	clone := &Collection{
		ApiVersion: c.ApiVersion,
		Kind:       c.Kind,
		MetaData:   c.MetaData,
		Spec: ColSpec{
			Requests: make([]Request, len(c.Spec.Requests)),
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
			ID:   "collection",
			Name: name,
		},
		Spec: ColSpec{
			Requests: make([]Request, 0),
		},
		FilePath: "",
	}
}
