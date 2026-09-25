package domain

const (
	ApiVersion = "v1"

	KindConfig      = "Config"
	KindWorkspace   = "Workspace"
	KindProtoFile   = "ProtoFile"
	KindEnv         = "Environment"
	KindRequest     = "Request"
	KindPreferences = "Preferences"
	KindCollection  = "Collection"
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
	// Secret keeps the value encrypted on disk and hidden in the UI.
	Secret bool `yaml:"secret,omitempty"`
	// Locked marks a secret value that could not be decrypted, because the
	// secret key is missing or locked. Value then still holds the ciphertext,
	// which is written back untouched. Runtime only.
	Locked bool `yaml:"-"`
}
