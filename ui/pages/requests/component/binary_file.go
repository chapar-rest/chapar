package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/mirzakhany/chapar/ui/theme"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type BinaryFile struct {
	selectFileButton widget.Clickable
	removeButton     widget.Clickable

	textField *widgets.TextField
	FileName  string

	onSelectFile func()
	onChanged    func(filePath string)
}

func NewBinaryFile(Filename string) *BinaryFile {
	bf := &BinaryFile{
		FileName:  Filename,
		textField: widgets.NewTextField(Filename, "File"),
	}

	bf.textField.Icon = widgets.FileFolderIcon
	bf.textField.IconPosition = widgets.IconPositionEnd
	bf.textField.SetMinWidth(200)
	return bf
}

func (b *BinaryFile) SetOnSelectFile(f func()) {
	b.onSelectFile = f
}

func (b *BinaryFile) SetOnChanged(f func(filePath string)) {
	b.onChanged = f
}

func (b *BinaryFile) SetFileName(name string) {
	b.FileName = name
	b.textField.SetText(name)
	if b.onChanged != nil {
		b.onChanged(name)
	}
}

func (b *BinaryFile) RemoveFile() {
	b.FileName = ""
	b.textField.SetText("")
}

func (b *BinaryFile) GetFilePath() string {
	return b.FileName
}

func (b *BinaryFile) Layout(gtx layout.Context, theme *theme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(32)
			gtx.Constraints.Max.X = gtx.Dp(250)
			return b.textField.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(32)
			if b.onSelectFile != nil && b.selectFileButton.Clicked(gtx) {
				b.onSelectFile()
			}

			return widgets.Button(theme.Material(), &b.selectFileButton, widgets.UploadIcon, widgets.IconPositionStart, "Select File").Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(32)
			if b.removeButton.Clicked(gtx) {
				b.RemoveFile()
				if b.onChanged != nil {
					b.onChanged("")
				}
			}

			return widgets.Button(theme.Material(), &b.removeButton, widgets.DeleteIcon, widgets.IconPositionStart, "Remove").Layout(gtx, theme)
		}),
	)
}
