package domain

import "fmt"

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

// CompareKeyValues compares two slices of KeyValue and returns true if they are equal
func CompareKeyValues(a, b []KeyValue) bool {
	if len(a) != len(b) {
		fmt.Println("CompareKeyValues length not equal")
		return false
	}

	for i, v := range a {
		if !CompareEnvValue(v, b[i]) {
			fmt.Println("CompareKeyValues not equal")
			return false
		}
	}

	return true
}
