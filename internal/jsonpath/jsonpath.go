package jsonpath

import (
	"context"
	"encoding/json"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
)

func Get(input string, path string) (interface{}, error) {
	builder := gval.Full(jsonpath.PlaceholderExtension())
	eval, err := builder.NewEvaluable(path)
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
