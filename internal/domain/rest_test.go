package domain

import "testing"

// TestParsePathParams is the test function for ParsePathParams
func TestParsePathParams(t *testing.T) {
	tests := []struct {
		name   string
		params string
		want   []KeyValue
	}{
		{
			name:   "Single key",
			params: "/{id}",
			want: []KeyValue{
				{Key: "id", Value: "", Enable: true},
			},
		},
		{
			name:   "Multiple keys",
			params: "/{user}/{project}/{repo}",
			want: []KeyValue{
				{Key: "user", Value: "", Enable: true},
				{Key: "project", Value: "", Enable: true},
				{Key: "repo", Value: "", Enable: true},
			},
		},
		{
			name:   "Nested or invalid keys",
			params: "/{{project}}/{repo}",
			want: []KeyValue{
				{Key: "repo", Value: "", Enable: true},
			},
		},
		{
			name:   "No keys",
			params: "/users/projects",
			want:   []KeyValue{},
		},
		{
			name:   "Empty string",
			params: "",
			want:   []KeyValue{},
		},
		{
			name:   "Keys with special characters",
			params: "/{key1}/{key_2}/{key-3}",
			want: []KeyValue{
				{Key: "key1", Value: "", Enable: true},
				{Key: "key_2", Value: "", Enable: true},
				{Key: "key-3", Value: "", Enable: true},
			},
		},
		{
			name:   "Multiple opening braces",
			params: "/{{id}}/{key}",
			want: []KeyValue{
				{Key: "key", Value: "", Enable: true},
			},
		},
		{
			name:   "Multiple closing braces",
			params: "/{id}}/{key}",
			want: []KeyValue{
				{Key: "id", Value: "", Enable: true},
				{Key: "key", Value: "", Enable: true},
			},
		},
		{
			name:   "Keys with invalid nesting",
			params: "/{user/{project}/{repo}",
			want: []KeyValue{
				{Key: "project", Value: "", Enable: true},
				{Key: "repo", Value: "", Enable: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePathParams(tt.params)

			if len(got) != len(tt.want) {
				t.Errorf("ParsePathParams() length = %v, want %v", len(got), len(tt.want))
			}

			for i, kv := range got {
				if kv.Key != tt.want[i].Key {
					t.Errorf("ParsePathParams()[%d].Key = %v, want %v", i, kv.Key, tt.want[i].Key)
				}
				if kv.Value != tt.want[i].Value {
					t.Errorf("ParsePathParams()[%d].Value = %v, want %v", i, kv.Value, tt.want[i].Value)
				}
				if kv.Enable != tt.want[i].Enable {
					t.Errorf("ParsePathParams()[%d].Enable = %v, want %v", i, kv.Enable, tt.want[i].Enable)
				}
			}
		})
	}
}
