package domain

import "strings"

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

// CompareKeyValues compares two slices of KeyValue and returns true if they are equal
func CompareKeyValues(a, b []KeyValue) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if !CompareEnvValue(v, b[i]) {
			return false
		}
	}

	return true
}

func KeyValuesToText(values []KeyValue) string {
	var text string
	for _, v := range values {
		text += v.Key + ": " + v.Value + "\n"
	}
	return text
}

func TextToKeyValue(txt string) []KeyValue {
	values := make([]KeyValue, 0)
	lines := strings.Split(txt, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		values = append(values, KeyValue{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}

	return values
}

func FindKeyValue(values []KeyValue, key string) string {
	for _, v := range values {
		if v.Key == key {
			return v.Value
		}
	}
	return ""
}
