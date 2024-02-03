package domain

type Environment struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Meta       EnvMeta    `yaml:"meta"`
	Values     []EnvValue `yaml:"values"`
}

type EnvMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type EnvValue struct {
	ID     string `yaml:"id"`
	Key    string `yaml:"key"`
	Value  string `yaml:"value"`
	Enable bool   `yaml:"enable"`
}
