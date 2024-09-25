package jsonpath

import "testing"

func Test_Get(t *testing.T) {
	t.Parallel()
	t.Run("Get value", func(t *testing.T) {
		const input = `{"foo": "bar"}`
		const path = "$.foo"

		v, err := Get(input, path)
		if err != nil {
			t.Fatalf("Get(%q, %q) = %v", input, path, err)
		}

		if v != "bar" {
			t.Fatalf("Get(%q, %q) = %v; want %v", input, path, v, "bar")
		}
	})

	t.Run("Get nested value", func(t *testing.T) {
		const input = `{"foo": {"bar": "baz"}}`
		const path = "$.foo.bar"

		v, err := Get(input, path)
		if err != nil {
			t.Fatalf("Get(%q, %q) = %v", input, path, err)
		}

		if v != "baz" {
			t.Fatalf("Get(%q, %q) = %v; want %v", input, path, v, "baz")
		}
	})

	t.Run("Get array value", func(t *testing.T) {
		const input = `{"foo": ["bar", "baz"]}`
		const path = "$.foo[0]"

		v, err := Get(input, path)
		if err != nil {
			t.Fatalf("Get(%q, %q) = %v", input, path, err)
		}

		if v != "bar" {
			t.Fatalf("Get(%q, %q) = %v; want %v", input, path, v, "bar")
		}
	})

	t.Run("GVAL filter", func(t *testing.T) {
		const input = `[{"key":"a","value" : "I"}, {"key":"b","value" : "II"}, {"key":"c","value" : "III"}]`
		const path = `$[? @.key=="b"].value`

		v, err := Get(input, path)
		if err != nil {
			t.Fatalf("Get(%q, %q) = %v", input, path, err)
		}

		if v == nil {
			t.Fatalf("Get(%q, %q) = %v; want %v", input, path, v, "not nil")
		}

		arr, ok := v.([]interface{})
		if !ok {
			t.Fatalf("Get(%q, %q) = %v; want %v", input, path, v, "[]interface{}")
		}

		if arr[0] != "II" {
			t.Fatalf("Get(%q, %q) = %v; want %v", input, path, v, "II")
		}
	})

	t.Run("Should fail if path is invalid", func(t *testing.T) {
		const input = `{"foo": "bar"}`
		const path = "$.foo."

		v, err := Get(input, path)
		if err == nil {
			t.Fatalf("Get(%q, %q) = %v; want error", input, path, v)
		}
	})

	t.Run("Should fail if input is invalid", func(t *testing.T) {
		const input = `{"foo": "bar"`
		const path = "$.foo"

		v, err := Get(input, path)
		if err == nil {
			t.Fatalf("Get(%q, %q) = %v; want error", input, path, v)
		}
	})

	t.Run("Should fail if input is not a JSON", func(t *testing.T) {
		const input = `foo`
		const path = "$.foo"

		v, err := Get(input, path)
		if err == nil {
			t.Fatalf("Get(%q, %q) = %v; want error", input, path, v)
		}
	})

	t.Run("Should return nil if path is not found", func(t *testing.T) {
		const input = `{"foo": "bar"}`
		const path = "$.baz"

		v, err := Get(input, path)
		if err == nil {
			t.Fatalf("Get(%q, %q) = %v; want error", input, path, v)
		}
	})
}
