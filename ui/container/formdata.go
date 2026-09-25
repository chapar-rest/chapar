package container

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
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
		cells := []ui.View{
			ui.Checkbox("form-en-"+f.ID, "").Check(f.Enable).OnToggle(func(v bool) {
				f.Enable = v
				markDirty()
			}),
			ui.Select("form-type-"+f.ID, typeOpts).Width(80).
				Selected(optionIndex(f.Type, typeOpts)).
				OnChange(func(v string) { f.Type = v; markDirty() }),
			ui.TextField("form-key-"+f.ID, f.Key).Placeholder("Key").Width(120).
				OnChange(func(s string) { f.Key = s; markDirty() }),
		}
		if f.Type == domain.FormFieldTypeFile {
			label := "Choose file…"
			if len(f.Files) > 0 {
				label = filepath.Base(f.Files[0])
				if len(f.Files) > 1 {
					label += fmt.Sprintf(" (+%d)", len(f.Files)-1)
				}
			}
			cells = append(cells, ui.Button("form-file-"+f.ID, ui.Text(label)).OnClick(func() {
				pickFiles(deps, func(paths []string) {
					f.Files = append(f.Files, paths...)
					markDirty()
					deps.WakeNow()
				})
			}).Grow(1))
		} else {
			cells = append(cells, AssistField(ui.TextField("form-val-"+f.ID, f.Value), VarSource(deps, nil)).
				Placeholder("Value").Grow(1).
				OnChange(func(s string) { f.Value = s; markDirty() }))
		}
		cells = append(cells, ui.IconButton("form-del-"+f.ID, icons.Trash2).OnClick(func() {
			*fields = append((*fields)[:i], (*fields)[i+1:]...)
			markDirty()
		}))
		rows = append(rows, ui.Row(cells...).Gap(th.Spacing.S))
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

// PickProtoFiles opens a file dialog that takes one or more .proto files.
func PickProtoFiles(deps Deps, onPick func([]string)) {
	if deps.Files == nil {
		return
	}
	fd := deps.Files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title:    "Add proto files",
		Mode:     ui.FileDialogOpenFile,
		Multiple: true,
		Filters:  []ui.FileFilter{{Label: "Proto", Exts: []string{".proto"}}},
		OnConfirm: func(paths []string) {
			if len(paths) > 0 {
				onPick(paths)
			}
		},
	})
}

// PickFolder opens a folder picker, for choosing an import root.
func PickFolder(deps Deps, title string, onPick func(string)) {
	if deps.Files == nil {
		return
	}
	fd := deps.Files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title: title,
		Mode:  ui.FileDialogOpenFolder,
		OnConfirm: func(paths []string) {
			if len(paths) > 0 {
				onPick(paths[0])
			}
		},
	})
}

// CertFileFilters limit a file dialog to certificate and key files.
var CertFileFilters = []ui.FileFilter{{Label: "Certificates", Exts: []string{".pem", ".crt", ".key"}}}

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
