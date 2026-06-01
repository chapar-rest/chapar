package environment

import (
	"log"
	"os"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/domain"
	appevents "github.com/chapar-rest/chapar/internal/events"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/pages/internal/listui"
	"github.com/chapar-rest/chapar/uiv2/widget"
	"github.com/google/uuid"
)

// Section is the environments sidebar section.
type Section struct {
	repo    repository.RepositoryV2
	tabView *widget.TabView
}

// New constructs the environments section.
func New(repo repository.RepositoryV2, tabView *widget.TabView) *Section {
	return &Section{repo: repo, tabView: tabView}
}

// MenuItem returns the side menu entry for environments.
func (s *Section) MenuItem() widget.SideMenuItem {
	return widget.SideMenuItem{Tag: "environments", Name: "Envs", Icon: icons.Menu}
}

// BuildListPanel renders the environments list in the left rail.
func (s *Section) BuildListPanel(panel *core.Frame) {
	listui.SectionHeader(panel, "Environments")

	actions := core.NewFrame(panel)
	actions.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.End
		st.Gap.Set(units.Dp(4))
		st.Padding.Set(units.Dp(2), units.Dp(0))
	})

	core.NewButton(actions).
		SetType(core.ButtonOutlined).
		SetText("Import").
		SetIcon(icons.FileOpen).
		OnClick(func(e events.Event) {
			s.openImportDialog(panel)
		})

	core.NewButton(actions).
		SetType(core.ButtonOutlined).
		SetText("New").
		SetIcon(icons.Add).
		OnClick(func(e events.Event) {
			s.createEnvironment(panel)
		})

	search := core.NewTextField(panel)
	search.SetPlaceholder("Search...")
	search.SetTrailingIcon(icons.Search)
	search.SendChangeOnInput()

	list := core.NewFrame(panel)
	list.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 1)
	})
	list.AddContextMenu(s.listPanelContextMenu(panel))

	populate := func(filter string) {
		list.DeleteChildren()
		environments, err := s.repo.LoadEnvironments()
		if err != nil {
			log.Println(err)
			core.NewText(list).
				SetType(core.TextBodyMedium).
				SetText("Failed to load environments")
			list.Update()
			return
		}
		if len(environments) == 0 {
			core.NewText(list).
				SetType(core.TextBodyMedium).
				SetText("No environments")
			list.Update()
			return
		}
		q := filter
		for _, env := range environments {
			env := env
			label := env.GetName()
			if label == "" {
				label = "Untitled environment"
			}
			if q != "" && !strings.Contains(strings.ToLower(label), strings.ToLower(q)) {
				continue
			}
			listui.ItemWith(list, label, env.MetaData.ID, listui.ItemOpts{
				OnClick: func() { s.openTab(env) },
				OnMenu: func(m *core.Scene) {
					core.NewButton(m).SetText("Duplicate").SetIcon(icons.ContentCopy).
						OnClick(func(e events.Event) {
							s.duplicateEnvironment(env, panel)
						})
					core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
						OnClick(func(e events.Event) {
							s.confirmDeleteEnvironment(env, panel)
						})
				},
			})
		}
		list.Update()
	}

	search.OnChange(func(e events.Event) {
		populate(search.Text())
	})

	populate("")
}

func (s *Section) refreshList(panel *core.Frame) {
	panel.DeleteChildren()
	s.BuildListPanel(panel)
	panel.Update()
}

func (s *Section) listPanelContextMenu(panel *core.Frame) func(m *core.Scene) {
	return func(m *core.Scene) {
		core.NewButton(m).SetText("New environment").SetIcon(icons.Add).
			OnClick(func(e events.Event) {
				s.createEnvironment(panel)
			})
		core.NewButton(m).SetText("Import...").SetIcon(icons.FileOpen).
			OnClick(func(e events.Event) {
				s.openImportDialog(panel)
			})
	}
}

func (s *Section) createEnvironment(panel *core.Frame) {
	env := domain.NewEnvironment("New Environment")
	if err := s.repo.CreateEnvironment(env); err != nil {
		log.Println(err)
		core.MessageDialog(panel, err.Error(), "Create failed")
		return
	}
	appevents.EnvironmentChangeTopic.Publish(env)
	s.refreshList(panel)
	s.openTab(env)
}

func (s *Section) duplicateEnvironment(env *domain.Environment, panel *core.Frame) {
	clone := env.Clone()
	name := clone.GetName()
	if name == "" {
		name = "Untitled environment"
	}
	clone.SetName(name + " (copy)")
	if err := s.repo.CreateEnvironment(clone); err != nil {
		log.Println(err)
		core.MessageDialog(panel, err.Error(), "Duplicate failed")
		return
	}
	appevents.EnvironmentChangeTopic.Publish(clone)
	s.refreshList(panel)
	s.openTab(clone)
}

func (s *Section) confirmDeleteEnvironment(env *domain.Environment, panel *core.Frame) {
	label := env.GetName()
	if label == "" {
		label = "Untitled environment"
	}
	d := core.NewBody("Delete environment")
	core.NewText(d).
		SetType(core.TextBodyMedium).
		SetText("Delete \"" + label + "\"? This cannot be undone.")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete").OnClick(func(e events.Event) {
			if err := s.repo.DeleteEnvironment(env); err != nil {
				log.Println(err)
				core.MessageDialog(d, err.Error(), "Delete failed")
				return
			}
			appevents.EnvironmentChangeTopic.Publish(nil)
			if s.tabView != nil {
				s.tabView.Close("env:" + env.MetaData.ID)
			}
			s.refreshList(panel)
			d.Close()
		})
	})
	d.RunDialog(panel)
}

func (s *Section) openTab(env *domain.Environment) {
	label := env.GetName()
	if label == "" {
		label = "Untitled environment"
	}
	key := "env:" + env.MetaData.ID
	s.tabView.Open(key, label, s.page(env, key))
}

func (s *Section) openImportDialog(panel *core.Frame) {
	d := core.NewBody("Import environment")
	fp := core.NewFilePicker(d).SetExtensions(".json")
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
			data, err := os.ReadFile(path)
			if err != nil {
				core.MessageDialog(d, err.Error(), "Import failed")
				return
			}
			if err := importer.ImportPostmanEnvironment(data, s.repo); err != nil {
				core.MessageDialog(d, err.Error(), "Import failed")
				return
			}
			appevents.EnvironmentChangeTopic.Publish(nil)
			s.refreshList(panel)
			d.Close()
		})
	})

	d.RunDialog(panel)
}

func (s *Section) page(env *domain.Environment, tabKey string) func(content *core.Frame) {
	return func(content *core.Frame) {
		content.Styler(func(st *styles.Style) {
			st.Direction = styles.Column
			st.Padding.Set(units.Dp(12))
			st.Gap.Set(units.Dp(6))
			st.Grow.Set(1, 1)
		})

		ensureKVIds := func() {
			for i := range env.Spec.Values {
				if env.Spec.Values[i].ID == "" {
					env.Spec.Values[i].ID = uuid.NewString()
				}
			}
		}

		save := func() {
			ensureKVIds()
			if err := s.repo.UpdateEnvironment(env); err != nil {
				log.Println("save environment:", err)
				core.MessageDialog(content, err.Error(), "Save failed")
				return
			}
			appevents.EnvironmentChangeTopic.Publish(env)
			if s.tabView != nil && tabKey != "" {
				s.tabView.SetTabLabel(tabKey, env.GetName())
			}
		}

		header := core.NewFrame(content)
		header.Styler(func(st *styles.Style) {
			st.Direction = styles.Row
			st.Align.Items = styles.Center
			st.Gap.Set(units.Dp(8))
			st.Grow.Set(1, 0)
			st.Min.X.Dp(0)
		})

		titleWrap := core.NewFrame(header)
		titleWrap.Styler(func(st *styles.Style) {
			st.Grow.Set(1, 0)
			st.Min.X.Dp(0)
			st.Overflow.X = styles.OverflowHidden
		})
		title := widget.NewEditableLabel(titleWrap, env.GetName())
		title.OnCommit(func(name string) {
			env.SetName(name)
			save()
		})

		searchWrap := core.NewFrame(header)
		searchWrap.Styler(func(st *styles.Style) {
			st.Grow.Set(0, 0)
			st.Min.X.Dp(160)
			st.Max.X.Dp(220)
		})
		search := core.NewTextField(searchWrap)
		search.SetPlaceholder("Search items")
		search.SetTrailingIcon(icons.Search)
		search.SendChangeOnInput()

		filter := ""

		caption := core.NewFrame(content)
		caption.Styler(func(st *styles.Style) {
			st.Direction = styles.Row
			st.Align.Items = styles.Center
			st.Justify.Content = styles.SpaceBetween
			st.Grow.Set(1, 0)
		})
		core.NewText(caption).
			SetType(core.TextBodySmall).
			SetText("Disabled items have no effect on your requests")

		tbl := widget.NewTable(content)
		tbl.OnDelete(func(int) { save() })
		tbl.SetSlice(&env.Spec.Values)
		tbl.SetTableStyler(func(w core.Widget, st *styles.Style, row, col int) {
			if filter == "" || row < 0 || row >= len(env.Spec.Values) {
				return
			}
			kv := env.Spec.Values[row]
			if !kvMatchesFilter(kv, filter) {
				w.AsWidget().SetState(true, states.Invisible)
			}
		})
		tbl.OnChange(func(e events.Event) {
			save()
		})

		addBtn := core.NewButton(caption).
			SetType(core.ButtonAction).
			SetIcon(icons.Add)
		addBtn.OnClick(func(e events.Event) {
			tbl.NewAt(-1)
			ensureKVIds()
			save()
		})

		search.OnChange(func(e events.Event) {
			filter = strings.TrimSpace(search.Text())
			tbl.Update()
		})
	}
}

func kvMatchesFilter(kv domain.KeyValue, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(kv.Key), q) ||
		strings.Contains(strings.ToLower(kv.Value), q)
}
