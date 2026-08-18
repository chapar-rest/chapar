package pages

import (
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/mirzakhany/yoga/ui"
)

type ProtoFiles struct {
	query string
	table *ui.Table
	repo  repository.RepositoryV2
	list  func() []*domain.ProtoFile
	load  func() error
	files *ui.FileDialog
	err   func(error)
}

func NewProtoFilesPage(repo repository.RepositoryV2, list func() []*domain.ProtoFile, load func() error, files *ui.FileDialog, errFn func(error)) *ProtoFiles {
	p := &ProtoFiles{repo: repo, list: list, load: load, files: files, err: errFn}
	p.table = ui.NewTable([]ui.TableColumn{
		{ID: "sel", Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: "name", Label: "Name", Kind: ui.TableColText, Width: 0, Sortable: true},
		{ID: "path", Label: "Path", Kind: ui.TableColText, Width: 0},
		{ID: "pkg", Label: "Package", Kind: ui.TableColText, Width: 160},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: "delete", Tooltip: "Delete"}})
	p.table.Actions[0].OnClick = func(rowID string) { p.deleteID(rowID) }
	p.table.Selectable = true
	p.table.MultiSelect = true
	p.Reload()
	return p
}

func (p *ProtoFiles) Reload() {
	files := p.list()
	rows := make([]ui.TableRow, 0, len(files))
	for _, f := range files {
		rows = append(rows, ui.TableRow{
			ID:    f.MetaData.ID,
			Icon:  "code",
			Cells: map[string]string{"name": f.MetaData.Name, "path": f.Spec.Path, "pkg": f.Spec.Package},
		})
	}
	p.table.SetRows(rows)
	p.table.SetFilter(p.query)
}

func (p *ProtoFiles) deleteID(id string) {
	for _, f := range p.list() {
		if f.MetaData.ID == id {
			if err := p.repo.DeleteProtoFile(f); err != nil {
				p.err(err)
				return
			}
			break
		}
	}
	_ = p.load()
	p.Reload()
}

func (p *ProtoFiles) deleteSelected() {
	for _, row := range p.table.Rows {
		if row.Selected {
			p.deleteID(row.ID)
		}
	}
}

func (p *ProtoFiles) add() {
	if p.files == nil {
		return
	}
	p.files.Show(ui.FileDialogOpts{
		Title:   "Add proto file",
		Mode:    ui.FileDialogOpenFile,
		Filters: []ui.FileFilter{{Label: "Proto", Exts: []string{".proto"}}},
		OnConfirm: func(paths []string) {
			for _, path := range paths {
				if err := importer.ImportProtoFileFromFile(path, p.repo); err != nil {
					pf := domain.NewProtoFile(filepath.Base(path))
					pf.Spec.Path = path
					if err2 := p.repo.CreateProtoFile(pf); err2 != nil {
						p.err(err)
						return
					}
				}
			}
			_ = p.load()
			p.Reload()
		},
	})
}

func (p *ProtoFiles) addImportPath() {
	if p.files == nil {
		return
	}
	p.files.Show(ui.FileDialogOpts{
		Title: "Add import path",
		Mode:  ui.FileDialogOpenFolder,
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			pf := domain.NewProtoFile(filepath.Base(paths[0]))
			pf.Spec.Path = paths[0]
			pf.Spec.IsImportPath = true
			if err := p.repo.CreateProtoFile(pf); err != nil {
				p.err(err)
				return
			}
			_ = p.load()
			p.Reload()
		},
	})
}

func (p *ProtoFiles) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Row(
			ui.Strong("Proto files"),
			ui.Spacer(),
			ui.TextField("proto-search", p.query).Placeholder("Search...").IconStart("search").Width(220).
				OnChange(func(s string) { p.query = s; p.table.SetFilter(s) }),
			ui.Button("proto-del", ui.Text("Delete selected")).OnClick(p.deleteSelected),
			ui.Button("proto-import-path", ui.Text("Import path")).OnClick(p.addImportPath),
			ui.Button("proto-add", ui.Text("Add")).Primary().IconStart("add").OnClick(p.add),
		).Gap(th.Spacing.S).Padding(th.Spacing.M),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(p.table).Grow(1),
	).Grow(1)
}
