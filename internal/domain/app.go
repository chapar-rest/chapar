package domain

const (
	ApiVersion = "v1"

	KindWorkspace     = "Workspace"
	KindEnv           = "Environment"
	KindRequest       = "Request"
	KindPreferences   = "Preferences"
	KindCollection    = "Collection"
	KindProtoFileList = "ProtoFileList"
)

type MetaData struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type KeyValue struct {
	ID     string `yaml:"id"`
	Key    string `yaml:"key"`
	Value  string `yaml:"value"`
	Enable bool   `yaml:"enable"`
}
