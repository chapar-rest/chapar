package jsonpath

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestQuery(t *testing.T) {
	var doc any
	if err := json.Unmarshal([]byte(`{"a":{"b":1},"arr":[{"k":1},{"k":2}],"n":null}`), &doc); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path  string
		want  any
		found bool
	}{
		{"$.a.b", 1.0, true},
		{"$.n", nil, true},
		{"$.arr[*].k", []any{1.0, 2.0}, true},
		{"$.x", nil, false},
		{"$.arr[5]", nil, false},
		{"$.a.b.c", nil, false},
	}
	for _, tt := range tests {
		got, found, err := Query(doc, tt.path)
		if err != nil {
			t.Fatalf("%s: %v", tt.path, err)
		}
		if found != tt.found || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s = %#v, %v; want %#v, %v", tt.path, got, found, tt.want, tt.found)
		}
	}

	if _, _, err := Query(doc, "$["); err == nil {
		t.Fatal("malformed path: want an error")
	}
}
