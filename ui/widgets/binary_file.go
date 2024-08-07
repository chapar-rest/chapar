package widgets

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type BinaryFile struct {
	refreshButton widget.Clickable

	textField *TextField
	FileName  string

	onSelectFile func()
	onChanged    func(filePath string)
	onRefresh    func()
}

func NewBinaryFile(filename string) *BinaryFile {
	bf := &BinaryFile{
		FileName:  filename,
		textField: NewTextField(filename, "File"),
	}

	bf.textField.SetText(filename)
	bf.textField.IconPosition = IconPositionEnd
	bf.textField.SetMinWidth(200)

	bf.updateIcon()
	return bf
}

func (b *BinaryFile) SetOnSelectFile(f func()) {
	b.onSelectFile = f
	b.textField.SetOnIconClick(func() {
		if b.FileName != "" {
			b.RemoveFile()
			if b.onChanged != nil {
				b.onChanged("")
			}
			return
		} else {
			// Select file
			f()
		}
	})
}

func (b *BinaryFile) SetOnChanged(f func(filePath string)) {
	b.onChanged = f
}

func (b *BinaryFile) SetOnRefresh(f func()) {
	b.onRefresh = f
}

func (b *BinaryFile) SetFileName(name string) {
	b.FileName = name
	b.textField.SetText(name)
	b.updateIcon()
	if b.onChanged != nil {
		b.onChanged(name)
	}
}

func (b *BinaryFile) RemoveFile() {
	b.FileName = ""
	b.textField.SetText("")
	b.updateIcon()
}

func (b *BinaryFile) GetFilePath() string {
	return b.FileName
}

func (b *BinaryFile) updateIcon() {
	if b.FileName != "" {
		b.textField.SetIcon(DeleteIcon, IconPositionEnd)
	} else {
		b.textField.SetIcon(UploadIcon, IconPositionEnd)
	}
}

func (b *BinaryFile) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(32)
			gtx.Constraints.Max.X = gtx.Dp(200)
			return b.textField.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(32)
			if b.refreshButton.Clicked(gtx) {
				if b.onRefresh != nil {
					b.onRefresh()
				}
			}

			if b.FileName == "" {
				return layout.Dimensions{}
			}

			btn := Button(theme.Material(), &b.refreshButton, RefreshIcon, IconPositionStart, "")
			btn.Inset = layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(4), Left: unit.Dp(6), Right: unit.Dp(2)}
			btn.Color = theme.ButtonTextColor
			return btn.Layout(gtx, theme)
		}),
	)
}
