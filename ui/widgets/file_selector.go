package widgets

import (
	"errors"
	"fmt"

	"gioui.org/layout"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
)

// FileSelector is a widget that allows the user to select a file. it handles the file selection and display the file name.
type FileSelector struct {
	textField *TextField
	FileName  string

	extensions []string

	explorer     *explorer.Explorer
	onSelectFile func()

	onChanged func(filePath string)
	changed   bool
}

func NewFileSelector(filename string, explorer *explorer.Explorer, extensions ...string) *FileSelector {
	bf := &FileSelector{
		FileName:   filename,
		textField:  NewTextField(filename, "File"),
		explorer:   explorer,
		extensions: extensions,
	}

	bf.textField.SetText(filename)
	bf.textField.IconPosition = IconPositionEnd
	bf.textField.SetMinWidth(200)
	bf.updateIcon()
	bf.SetOnSelectFile(bf.handleExplorerSelect)
	return bf
}

func (b *FileSelector) SetExplorer(explorer *explorer.Explorer) {
	b.explorer = explorer
}

func (b *FileSelector) handleExplorerSelect() {
	if b.explorer == nil {
		return
	}

	b.explorer.ChoseFile(func(result explorer.Result) {
		if result.Error != nil {
			if !errors.Is(result.Error, explorer.ErrUserDecline) {
				fmt.Printf("failed to get file, %s\n", result.Error)
				return
			}
			return
		}
		if result.FilePath == "" {
			return
		}

		b.SetFileName(result.FilePath)
		b.changed = true
	}, b.extensions...)
}

func (b *FileSelector) SetOnSelectFile(f func()) {
	b.onSelectFile = f
	b.textField.SetOnIconClick(func() {
		if b.FileName != "" {
			b.RemoveFile()
			return
		} else {
			// Select file
			f()
			b.onChangeCallback()
		}
	})
}

func (b *FileSelector) SetFileName(name string) {
	b.FileName = name
	b.textField.SetText(name)
	b.updateIcon()
	b.changed = true
	b.onChangeCallback()
}

func (b *FileSelector) Changed() bool {
	out := b.changed
	b.changed = false
	return out
}

func (b *FileSelector) RemoveFile() {
	b.FileName = ""
	b.textField.SetText("")
	b.updateIcon()
	b.changed = true
	b.onChangeCallback()
}

func (b *FileSelector) onChangeCallback() {
	if b.onChanged != nil {
		b.onChanged(b.FileName)
	}
}

func (b *FileSelector) SetOnChanged(f func(filePath string)) {
	b.onChanged = f
}

func (b *FileSelector) GetFilePath() string {
	return b.FileName
}

func (b *FileSelector) updateIcon() {
	if b.FileName != "" {
		b.textField.SetIcon(DeleteIcon, IconPositionEnd)
	} else {
		b.textField.SetIcon(UploadIcon, IconPositionEnd)
	}
}

func (b *FileSelector) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Max.Y = gtx.Dp(32)
	gtx.Constraints.Max.X = gtx.Dp(200)
	return b.textField.Layout(gtx, theme)
}
