package container

import (
	"fmt"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// FormDataPane renders multipart form fields.
func FormDataPane(th *theme.Theme, id string, fields *[]domain.FormField, deps Deps, markDirty func()) ui.View {
	if *fields == nil {
		*fields = []domain.FormField{}
	}
	rows := []ui.View{
		ui.Row(
			ui.Text("Form fields"),
			ui.Spacer(),
			ui.IconButton("form-add-"+id, icons.Plus).OnClick(func() {
				*fields = append(*fields, domain.FormField{
					ID:     uuid.NewString(),
					Type:   domain.FormFieldTypeText,
					Enable: true,
				})
				markDirty()
			}),
		).PaddingXY(0, th.Spacing.S),
	}
	for i := range *fields {
		i := i
		f := &(*fields)[i]
		typeOpts := []ui.SelectOption{
			{Label: "Text", Value: domain.FormFieldTypeText},
			{Label: "File", Value: domain.FormFieldTypeFile},
		}
		row := ui.Row(
			ui.Checkbox("form-en-"+f.ID, "").Check(f.Enable).OnToggle(func(v bool) {
				f.Enable = v
				markDirty()
			}),
			ui.Select("form-type-"+f.ID, typeOpts).Width(80).
				Selected(optionIndex(f.Type, typeOpts)).
				OnChange(func(v string) { f.Type = v; markDirty() }),
			ui.TextField("form-key-"+f.ID, f.Key).Placeholder("Key").Width(120).
				OnChange(func(s string) { f.Key = s; markDirty() }),
		).Gap(th.Spacing.S).Align(ui.AlignCenter)
		if f.Type == domain.FormFieldTypeFile {
			label := "Choose file…"
			if len(f.Files) > 0 {
				label = filepath.Base(f.Files[0])
				if len(f.Files) > 1 {
					label += fmt.Sprintf(" (+%d)", len(f.Files)-1)
				}
			}
			row = ui.Row(row,
				ui.Button("form-file-"+f.ID, ui.Text(label)).OnClick(func() {
					pickFiles(deps, func(paths []string) {
						f.Files = append(f.Files, paths...)
						markDirty()
						deps.WakeNow()
					})
				}),
			).Gap(th.Spacing.S).Align(ui.AlignCenter)
		} else {
			row = ui.Row(row,
				ui.TextField("form-val-"+f.ID, f.Value).Placeholder("Value").Grow(1).
					OnChange(func(s string) { f.Value = s; markDirty() }),
			).Gap(th.Spacing.S).Align(ui.AlignCenter)
		}
		row = ui.Row(row,
			ui.IconButton("form-del-"+f.ID, icons.Trash2).OnClick(func() {
				*fields = append((*fields)[:i], (*fields)[i+1:]...)
				markDirty()
			}),
		).Gap(th.Spacing.S).Align(ui.AlignCenter)
		rows = append(rows, row)
	}
	return ui.Column(
		ui.Scroll("form-scroll-"+id, ui.Column(rows...).Gap(th.Spacing.S)),
	).Grow(1)
}

func pickFiles(deps Deps, onPick func([]string)) {
	if deps.Files == nil {
		return
	}
	fd := deps.Files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title: "Choose file",
		Mode:  ui.FileDialogOpenFile,
		OnConfirm: func(paths []string) {
			if len(paths) > 0 {
				onPick(paths)
			}
		},
	})
}

// PickProtoFile opens a file dialog for .proto files.
func PickProtoFile(deps Deps, onPick func(string)) {
	pickSingleFile(deps, "Choose proto file", []string{".proto"}, onPick)
}

// PickCertFile opens a file dialog for certificate files.
func PickCertFile(deps Deps, onPick func(string)) {
	pickSingleFile(deps, "Choose certificate", []string{".pem", ".crt", ".key"}, onPick)
}

func pickSingleFile(deps Deps, title string, exts []string, onPick func(string)) {
	if deps.Files == nil {
		return
	}
	fd := deps.Files()
	if fd == nil {
		return
	}
	opts := ui.FileDialogOpts{
		Title: title,
		Mode:  ui.FileDialogOpenFile,
		OnConfirm: func(paths []string) {
			if len(paths) > 0 {
				onPick(paths[0])
			}
		},
	}
	if len(exts) > 0 {
		opts.Filters = []ui.FileFilter{{Label: "Files", Exts: exts}}
	}
	fd.Show(opts)
}

// BinaryFilePicker renders a file picker for binary request bodies.
func BinaryFilePicker(th *theme.Theme, id, path string, deps Deps, onChange func(string)) ui.View {
	label := path
	if label == "" {
		label = "Choose file…"
	} else {
		label = filepath.Base(path)
	}
	return ui.Column(
		ui.Button("binary-pick-"+id, ui.Text(label)).IconStart(icons.Upload).OnClick(func() {
			pickSingleFile(deps, "Choose binary file", nil, onChange)
		}),
		ui.Caption(path).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
	).Gap(th.Spacing.S)
}
