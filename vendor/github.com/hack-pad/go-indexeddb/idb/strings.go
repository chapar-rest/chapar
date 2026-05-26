//go:build js && wasm
// +build js,wasm

package idb

import (
	"github.com/hack-pad/safejs"
)

func sliceFromStrings(strs []string) []interface{} {
	values := make([]interface{}, 0, len(strs))
	for _, s := range strs {
		values = append(values, s)
	}
	return values
}

func stringsFromArray(arr safejs.Value) ([]string, error) {
	var strs []string
	iterErr := iterArray(arr, func(i int, value safejs.Value) (bool, error) {
		str, err := value.String()
		if err != nil {
			return false, err
		}
		strs = append(strs, str)
		return true, nil
	})
	return strs, iterErr
}

func iterArray(arr safejs.Value, visit func(i int, value safejs.Value) (keepGoing bool, visitErr error)) (err error) {
	length, err := arr.Length()
	if err != nil {
		return err
	}
	for i := 0; i < length; i++ {
		index, err := arr.Index(i)
		if err != nil {
			return err
		}
		keepGoing, visitErr := visit(i, index)
		if !keepGoing || visitErr != nil {
			return visitErr
		}
	}
	return nil
}
