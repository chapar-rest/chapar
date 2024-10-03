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
	var values []KeyValue
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
