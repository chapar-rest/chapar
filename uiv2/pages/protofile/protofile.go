package protofile

import (
	"log"
	"path/filepath"
	"sort"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress/grpc"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/widget"
)

const (
	pagePadTop     = 30
	pagePadSide    = 16
	searchWidth    = 200
	gapBeforeTable = 16
)

type protoRow struct {
	Selected bool   `label:" "`
	Path     string `label:"Path"`
	Package  string `label:"Package"`
	Services string `label:"Services"`
	ID       string `display:"-"`
}

// Section is the proto files sidebar section.
type Section struct {
	repo repository.RepositoryV2
}

// New constructs the proto files section.
func New(repo repository.RepositoryV2) *Section {
	return &Section{repo: repo}
}

// MenuItem returns the side menu entry for proto files.
func (s *Section) MenuItem() widget.SideMenuItem {
	return widget.SideMenuItem{Tag: "protofiles", Name: "Proto", Icon: icons.Folder}
}

// BuildListPanel is a no-op; proto files render as a full page.
func (s *Section) BuildListPanel(_ *core.Frame) {}

// BuildContent renders the proto files management page in the main content area.
func (s *Section) BuildContent(content *core.Frame) {
	content.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 1)
		st.Padding.Set(units.Dp(pagePadTop), units.Dp(pagePadSide))
	})

	top := core.NewFrame(content)
	top.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.Start
		st.Justify.Content = styles.SpaceBetween
		st.Gap.Set(units.Dp(16))
		st.Grow.Set(1, 0)
	})

	leftCol := core.NewFrame(top)
	leftCol.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 0)
		st.Gap.Set(units.Dp(5))
	})
	core.NewText(leftCol).
		SetType(core.TextTitleLarge).
		SetText("Proto files and Import Paths")
	core.NewText(leftCol).
		SetType(core.TextBodyMedium).
		SetText("Manage proto files as dependencies and import paths").
		Styler(func(st *styles.Style) {
			st.Color = colors.Scheme.OnSurfaceVariant
		})

	actionsRow := core.NewFrame(top)
	actionsRow.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.Center
		st.Gap.Set(units.Dp(5))
		st.Grow.Set(0, 0)
	})

	deleteSelectedBtn := core.NewButton(actionsRow).
		SetType(core.ButtonTonal).
		SetText("Delete Selected").
		SetIcon(icons.Delete)
	deleteSelectedBtn.SetState(true, states.Invisible)

	addImportBtn := core.NewButton(actionsRow).
		SetType(core.ButtonTonal).
		SetText("Add Import Path").
		SetIcon(icons.Folder)

	addProtoBtn := core.NewButton(actionsRow).
		SetType(core.ButtonTonal).
		SetText("Add Proto file").
		SetIcon(icons.Add)

	searchWrap := core.NewFrame(actionsRow)
	searchWrap.Styler(func(st *styles.Style) {
		st.Min.X.Dp(searchWidth)
		st.Max.X.Dp(searchWidth)
		st.Padding.Set(units.Dp(0), units.Dp(0), units.Dp(0), units.Dp(25))
	})
	search := core.NewTextField(searchWrap)
	search.SetPlaceholder("Search...")
	search.SetTrailingIcon(icons.Search)
	search.SendChangeOnInput()

	tableArea := core.NewFrame(content)
	tableArea.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 1)
		st.Padding.Set(units.Dp(gapBeforeTable), units.Dp(0), units.Dp(0), units.Dp(0))
	})

	tableWrap := core.NewFrame(tableArea)
	tableWrap.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 1)
		st.Background = colors.Scheme.SurfaceContainerLow
		st.Border.Style.Set(styles.BorderSolid)
		st.Border.Width.Set(units.Dp(1))
		st.Border.Color.Set(colors.Scheme.OutlineVariant)
		st.Border.Radius = styles.BorderRadiusSmall
		st.Overflow.Y = styles.OverflowAuto
	})

	headerRow := core.NewFrame(tableWrap)
	headerRow.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.Center
		st.Grow.Set(1, 0)
		st.Padding.Set(units.Dp(8), units.Dp(6))
		st.Background = colors.Scheme.SurfaceContainerHigh
		st.Border.Style.Bottom = styles.BorderSolid
		st.Border.Width.Bottom = units.Dp(1)
		st.Border.Color.Bottom = colors.Scheme.OutlineVariant
	})
	headerSelectCol := core.NewFrame(headerRow)
	headerSelectCol.Styler(func(st *styles.Style) {
		st.Min.X.Dp(44)
		st.Max.X.Dp(44)
		st.Align.Items = styles.Center
		st.Justify.Content = styles.Center
	})
	headerSelectAll := core.NewSwitch(headerSelectCol)
	headerSelectAll.SetType(core.SwitchCheckbox)
	for _, label := range []struct {
		text string
		grow float32
		min  float32
	}{
		{"Path", 3, 120},
		{"Package", 1, 80},
		{"Services", 1, 80},
	} {
		col := core.NewFrame(headerRow)
		col.Styler(func(st *styles.Style) {
			st.Grow.Set(label.grow, 0)
			st.Min.X.Dp(label.min)
		})
		core.NewText(col).
			SetType(core.TextBodyMedium).
			SetText(label.text).
			Styler(func(st *styles.Style) {
				st.SetTextWrap(false)
			})
	}
	headerActionCol := core.NewFrame(headerRow)
	headerActionCol.Styler(func(st *styles.Style) {
		st.Min.X.Dp(44)
		st.Max.X.Dp(44)
	})

	var (
		rows       []protoRow
		protoFiles []*domain.ProtoFile
		filter     string
	)

	tbl := widget.NewTable(tableWrap)
	tbl.SetSlice(&rows)
	tbl.Styler(func(st *styles.Style) {
		st.Grow.Set(1, 1)
	})
	tree.AddChildInit(tbl, "header", func(h *core.Frame) {
		h.SetState(true, states.Invisible)
	})

	updateDeleteSelected := func() {
		show := false
		for _, r := range rows {
			if r.Selected {
				show = true
				break
			}
		}
		deleteSelectedBtn.SetState(!show, states.Invisible)
		deleteSelectedBtn.Update()
	}

	refresh := func() {
		loaded, err := s.repo.LoadProtoFiles()
		if err != nil {
			log.Println(err)
			core.MessageDialog(content, err.Error(), "Load failed")
			return
		}

		sort.Slice(loaded, func(i, j int) bool {
			return loaded[i].Spec.Path < loaded[j].Spec.Path
		})

		protoFiles = loaded
		rows = make([]protoRow, 0, len(loaded))
		for _, pf := range loaded {
			rows = append(rows, protoRow{
				Path:     pf.Spec.Path,
				Package:  pf.Spec.Package,
				Services: strings.Join(pf.Spec.Services, ","),
				ID:       pf.MetaData.ID,
			})
		}

		headerSelectAll.SetChecked(false)
		tbl.Update()
		updateDeleteSelected()
	}

	tbl.SetTableStyler(func(w core.Widget, st *styles.Style, row, col int) {
		if filter == "" || row < 0 || row >= len(rows) {
			return
		}
		r := rows[row]
		if !rowMatchesFilter(r, filter) {
			w.AsWidget().SetState(true, states.Invisible)
		}
	})

	tbl.OnDelete(func(row int) {
		if row < 0 || row >= len(protoFiles) {
			return
		}
		pf := protoFiles[row]
		if err := s.repo.DeleteProtoFile(pf); err != nil {
			log.Println(err)
			core.MessageDialog(content, err.Error(), "Delete failed")
			refresh()
			return
		}
		protoFiles = append(protoFiles[:row], protoFiles[row+1:]...)
		updateDeleteSelected()
	})

	tbl.OnChange(func(e events.Event) {
		updateDeleteSelected()
	})

	headerSelectAll.OnChange(func(e events.Event) {
		checked := headerSelectAll.IsChecked()
		q := strings.ToLower(strings.TrimSpace(filter))
		for i := range rows {
			if q != "" && !rowMatchesFilter(rows[i], q) {
				continue
			}
			rows[i].Selected = checked
		}
		tbl.Update()
		updateDeleteSelected()
	})

	deleteSelectedBtn.OnClick(func(e events.Event) {
		toDelete := make([]*domain.ProtoFile, 0)
		remainingRows := make([]protoRow, 0, len(rows))
		remainingProtos := make([]*domain.ProtoFile, 0, len(protoFiles))

		for i, r := range rows {
			if r.Selected {
				toDelete = append(toDelete, protoFiles[i])
				continue
			}
			remainingRows = append(remainingRows, r)
			remainingProtos = append(remainingProtos, protoFiles[i])
		}

		for _, pf := range toDelete {
			if err := s.repo.DeleteProtoFile(pf); err != nil {
				log.Println(err)
				core.MessageDialog(content, err.Error(), "Delete failed")
				refresh()
				return
			}
		}

		rows = remainingRows
		protoFiles = remainingProtos
		headerSelectAll.SetChecked(false)
		tbl.Update()
		updateDeleteSelected()
	})

	search.OnChange(func(e events.Event) {
		filter = strings.TrimSpace(search.Text())
		tbl.Update()
	})

	addImportBtn.OnClick(func(e events.Event) {
		s.openImportPathDialog(content, refresh)
	})
	addProtoBtn.OnClick(func(e events.Event) {
		s.openAddProtoDialog(content, refresh)
	})

	refresh()
}

func rowMatchesFilter(r protoRow, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(r.Path), q) ||
		strings.Contains(strings.ToLower(r.Package), q) ||
		strings.Contains(strings.ToLower(r.Services), q)
}

func (s *Section) openImportPathDialog(content *core.Frame, refresh func()) {
	d := core.NewBody("Add Import Path")
	fp := core.NewFilePicker(d)
	fp.Filterer = core.FilePickerDirOnlyFilter
	fp.Styler(func(st *styles.Style) {
		st.Grow.Set(1, 1)
		st.Min.Set(units.Dp(480), units.Dp(320))
	})

	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			path := strings.TrimSpace(fp.SelectedFile())
			if path == "" {
				return
			}
			proto := domain.NewProtoFile(filepath.Base(path))
			proto.Spec.IsImportPath = true
			proto.Spec.Path = path
			if err := s.repo.CreateProtoFile(proto); err != nil {
				core.MessageDialog(d, err.Error(), "Create failed")
				return
			}
			d.Close()
			refresh()
		})
	})

	d.RunDialog(content)
}

func (s *Section) openAddProtoDialog(content *core.Frame, refresh func()) {
	d := core.NewBody("Add Proto file")
	fp := core.NewFilePicker(d).SetExtensions(".proto")
	fp.Styler(func(st *styles.Style) {
		st.Grow.Set(1, 1)
		st.Min.Set(units.Dp(480), units.Dp(320))
	})

	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			path := fp.SelectedFile()
			if path == "" {
				return
			}
			existing, err := s.repo.LoadProtoFiles()
			if err != nil {
				core.MessageDialog(d, err.Error(), "Load failed")
				return
			}

			fileName := filepath.Base(path)
			proto := domain.NewProtoFile(fileName)
			pkg, services, err := parseProtoInfo(existing, path)
			if err != nil {
				core.MessageDialog(d, err.Error(), "Parse failed")
				return
			}
			proto.Spec.Path = path
			proto.Spec.Package = pkg
			proto.Spec.Services = services

			if err := s.repo.CreateProtoFile(proto); err != nil {
				core.MessageDialog(d, err.Error(), "Create failed")
				return
			}
			d.Close()
			refresh()
		})
	})

	d.RunDialog(content)
}

func parseProtoInfo(existing []*domain.ProtoFile, filePath string) (string, []string, error) {
	importPaths, fileNames := grpc.GetImportPaths(existing, []string{filePath})
	pInfo, err := grpc.ProtoFilesFromDisk(importPaths, fileNames)
	if err != nil {
		return "", nil, err
	}

	var pkg string
	var services []string
	pInfo.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		pkg = string(f.Package())
		for i := 0; i < f.Services().Len(); i++ {
			services = append(services, string(f.Services().Get(i).FullName()))
		}
		return true
	})

	return pkg, services, nil
}
