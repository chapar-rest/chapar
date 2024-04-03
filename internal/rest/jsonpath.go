package rest

import (
	"encoding/json"

	"github.com/PaesslerAG/jsonpath"
)

func GetJSONPATH(input string, path string) (interface{}, error) {
	v := interface{}(nil)
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return nil, err
	}

	pathData, err := jsonpath.Get(path, v)
	if err != nil {
		return nil, err
	}

	if pathData == nil {
		return nil, nil
	}

	return pathData, nil
}
