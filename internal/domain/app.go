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
	KindTestCase    = "TestCase"
	KindTestBundle  = "TestBundle"
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
	// Unset marks a secret whose value is not stored on this machine, as in a
	// workspace freshly cloned from git: the shared file names the secret but
	// leaves its value out. Runtime only.
	Unset bool `yaml:"-"`
}

// Usable reports whether the value takes part in a request: it is turned on,
// and it is available (substituting a locked value would send ciphertext, an
// unset one an empty string).
func (kv KeyValue) Usable() bool {
	return kv.Enable && kv.Available()
}

// Available reports whether the value can be read on this machine: it is not
// a secret that is still encrypted or missing.
func (kv KeyValue) Available() bool {
	return !kv.Locked && !kv.Unset
}
