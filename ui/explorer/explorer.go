package explorer

import (
	"fmt"
	"io"
	"os"

	"gioui.org/app"
	"gioui.org/x/explorer"
)

var (
	ErrUserDecline = explorer.ErrUserDecline
)

type Explorer struct {
	expl *explorer.Explorer
	w    *app.Window
}

type Result struct {
	Data     []byte
	Error    error
	FilePath string
}

func NewExplorer(w *app.Window) *Explorer {
	return &Explorer{
		expl: explorer.NewExplorer(w),
		w:    w,
	}
}

func (e *Explorer) ChoseFile(onResult func(r Result) error, extensions ...string) <-chan error {
	errChan := make(chan error)
	fn := func(onResult func(r Result) error) {
		defer func(e *Explorer) {
			e.w.Invalidate()
		}(e)

		file, err := e.expl.ChooseFile(extensions...)
		if err != nil {
			err = fmt.Errorf("failed opening file: %w", err)
			errChan <- onResult(Result{Error: err})
			return
		}

		defer func(file io.ReadCloser) {
			err := file.Close()
			if err != nil {
				err = fmt.Errorf("failed closing file: %w", err)
				errChan <- onResult(Result{Error: err})
			}
		}(file)

		filePath := ""
		// get file path if possible
		if f, ok := file.(*os.File); ok {
			filePath = f.Name()
		}

		data, err := io.ReadAll(file)
		if err != nil {
			err = fmt.Errorf("failed reading file: %w", err)
			errChan <- onResult(Result{Error: err, FilePath: filePath})
			return
		}
		errChan <- onResult(Result{Data: data, FilePath: filePath, Error: nil})
	}

	go func() {
		fn(onResult)
	}()

	return errChan
}
