package util

import (
	"bytes"
	"encoding/json"
)

const jsonIndent = "    "

func IsJSON(s string) bool {
	return IsJSONBytes([]byte(s))
}

// IsJSONBytes reports whether data is valid JSON without copying it into a
// string or building a value tree.
func IsJSONBytes(data []byte) bool {
	return json.Valid(data)
}

// PrettyJSON indents data for display without decoding it.
//
// This shows the response as the server sent it: object keys keep their
// original order, numbers keep their original text, and escape sequences are
// left as-is. The previous implementation decoded into interface{} and
// marshalled back, which reordered keys alphabetically (Go map marshalling) and
// round-tripped every number through float64 — so an int64 id like
// 9007199254740993 displayed as ...992. It also cost many times the body size
// in live objects: a 10 MB response pushed the Go heap from 45 MB to 195 MB,
// and the process never gave that back.
//
// The buffer is pre-sized so it does not grow by doubling: an undersized guess
// on a 20 MB result churns through tens of MB of intermediate arrays before it
// settles. Four-space indenting of compact JSON roughly doubles it, so budget
// for that up front — overshooting costs one oversized array, undershooting
// costs a full realloc and copy.
func PrettyJSON(data []byte) (string, error) {
	// json.Indent copies leading and trailing whitespace from src through to
	// dst, which would show up as blank lines around the body. Trimming is a
	// subslice, not a copy.
	data = bytes.TrimSpace(data)

	var out bytes.Buffer
	out.Grow(2*len(data) + 1024)
	if err := json.Indent(&out, data, "", jsonIndent); err != nil {
		return "", err
	}
	return out.String(), nil
}

func ParseJSON(text string) (map[string]any, error) {
	var js map[string]any
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		return nil, err
	}
	return js, nil
}

func EncodeJSON(data any) ([]byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return b, nil
}
