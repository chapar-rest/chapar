package explorer

import (
	"fmt"
	"io"

	"gioui.org/app"
	"gioui.org/x/explorer"
)

type Explorer struct {
	expl *explorer.Explorer
	w    *app.Window
}

type Result struct {
	Data  []byte
	Error error
}

func NewExplorer(w *app.Window) *Explorer {
	return &Explorer{
		expl: explorer.NewExplorer(w),
		w:    w,
	}
}

func (e *Explorer) ChoseFiles(onResult func(r Result), extensions ...string) {
	go func(onResult func(r Result)) {
		defer func(e *Explorer) {
			e.w.Invalidate()
		}(e)

		file, err := e.expl.ChooseFile(extensions...)
		if err != nil {
			err = fmt.Errorf("failed opening file: %w", err)
			onResult(Result{Error: err})
			return
		}

		defer func(file io.ReadCloser) {
			err := file.Close()
			if err != nil {
				err = fmt.Errorf("failed closing file: %w", err)
				onResult(Result{Error: err})
			}
		}(file)

		data, err := io.ReadAll(file)
		if err != nil {
			err = fmt.Errorf("failed reading file: %w", err)
			onResult(Result{Error: err})
			return
		}
		onResult(Result{Data: data, Error: nil})
	}(onResult)
}
