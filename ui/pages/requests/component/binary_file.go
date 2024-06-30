package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type FileSelector struct {
	refreshButton widget.Clickable

	textField *widgets.TextField
	FileName  string

	onSelectFile func()
	onChanged    func(filePath string)
	onRefresh    func()
}

func NewFileSelector(filename string) *FileSelector {
	bf := &FileSelector{
		FileName:  filename,
		textField: widgets.NewTextField(filename, "File"),
	}

	bf.textField.SetText(filename)
	bf.textField.IconPosition = widgets.IconPositionEnd
	bf.textField.SetMinWidth(200)

	bf.updateIcon()
	return bf
}

func (b *FileSelector) SetOnSelectFile(f func()) {
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

func (b *FileSelector) SetOnChanged(f func(filePath string)) {
	b.onChanged = f
}

func (b *FileSelector) SetOnRefresh(f func()) {
	b.onRefresh = f
}

func (b *FileSelector) SetFileName(name string) {
	b.FileName = name
	b.textField.SetText(name)
	b.updateIcon()
	if b.onChanged != nil {
		b.onChanged(name)
	}
}

func (b *FileSelector) RemoveFile() {
	b.FileName = ""
	b.textField.SetText("")
	b.updateIcon()
}

func (b *FileSelector) GetFilePath() string {
	return b.FileName
}

func (b *FileSelector) updateIcon() {
	if b.FileName != "" {
		b.textField.SetIcon(widgets.DeleteIcon, widgets.IconPositionEnd)
	} else {
		b.textField.SetIcon(widgets.UploadIcon, widgets.IconPositionEnd)
	}
}

func (b *FileSelector) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
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

			btn := widgets.Button(theme.Material(), &b.refreshButton, widgets.RefreshIcon, widgets.IconPositionStart, "")
			btn.Inset = layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(4), Left: unit.Dp(6), Right: unit.Dp(2)}
			btn.Color = theme.ButtonTextColor
			return btn.Layout(gtx, theme)
		}),
	)
}
