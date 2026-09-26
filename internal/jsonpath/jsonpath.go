package jsonpath

import (
	"context"
	"encoding/json"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
)

func Get(input string, path string) (interface{}, error) {
	eval, err := compile(path)
	if err != nil {
		return nil, err
	}

	v := interface{}(nil)
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return nil, err
	}

	pathData, err := eval(context.Background(), v)
	if err != nil {
		return nil, err
	}

	if pathData == nil {
		return nil, nil
	}

	return pathData, nil
}

// Query evaluates path against an already decoded JSON value. A path that
// matches nothing (an unknown key, an index out of range, a key on a
// scalar) reports found=false without an error; only a malformed path is an
// error. A JSON null that is there is found, with a nil result.
func Query(v any, path string) (result any, found bool, err error) {
	eval, err := compile(path)
	if err != nil {
		return nil, false, err
	}
	result, err = eval(context.Background(), v)
	if err != nil {
		return nil, false, nil
	}
	return result, true, nil
}

// Valid reports whether path parses.
func Valid(path string) error {
	_, err := compile(path)
	return err
}

func compile(path string) (gval.Evaluable, error) {
	return gval.Full(jsonpath.PlaceholderExtension()).NewEvaluable(path)
}
