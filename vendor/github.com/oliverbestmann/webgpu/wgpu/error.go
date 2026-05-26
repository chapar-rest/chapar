package wgpu

import "fmt"

type Error struct {
	Context string
	Wrapped error
}

func (v Error) Error() string {
	if v.Context == "" {
		return v.Wrapped.Error()
	}

	return fmt.Sprintf("%s: %s", v.Context, v.Wrapped.Error())
}

func (v Error) Unwrap() error {
	return v.Wrapped
}

func panicIf(err error, context string) {
	if err != nil {
		panic(err)
	}
}

func wrap(err error, context string) error {
	return Error{
		Context: context,
		Wrapped: err,
	}
}
