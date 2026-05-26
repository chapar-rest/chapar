package settings

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

type section struct {
	id    string
	title string
	icon  icons.Icon
}

var sections = []section{
	{
		id:    "general",
		title: "General",
		icon:  icons.Tune,
	},
	{
		id:    "scripting",
		title: "Scripting",
		icon:  icons.Code,
	},
	{
		id:    "editor",
		title: "Editor",
		icon:  icons.Edit,
	},
	{
		id:    "data",
		title: "Data",
		icon:  icons.Folder,
	},
	{
		id:    "appearance",
		title: "Appearance",
		icon:  icons.Palette,
	},
}

type dialogState struct {
	draft              Data
	dirty              bool
	sectionID          string
	sectionList        *core.Frame
	nav                *core.Frame
	savedWorkspacePath string
}

var settingsDialog = &dialogState{sectionID: "appearance"}

// OpenDialog opens the Chapar settings dialog.
func OpenDialog(ctx core.Widget) {
	if core.RecycleDialog(settingsDialog) {
		return
	}

	settingsDialog.draft = *App
	settingsDialog.dirty = false
	settingsDialog.sectionID = sections[0].id
	settingsDialog.savedWorkspacePath = App.Data.WorkspacePath

	d := core.NewBody("Settings")
	d.SetTitle("Settings")
	d.SetData(settingsDialog)
	d.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(640), units.Dp(420))
		s.Max.Set(units.Dp(800), units.Dp(560))
	})
	d.FinalStyler(func(s *styles.Style) {
		s.Padding.Set(units.Dp(4))
	})
	buildSettingsBody(d)

	d.AddBottomBar(func(bar *core.Frame) {
		bar.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Justify.Content = styles.End
			s.Gap.Set(units.Dp(8))
		})

		core.NewButton(bar).
			SetType(core.ButtonText).
			SetText("Load defaults").
			OnClick(func(e events.Event) {
				settingsDialog.draft = defaultData()
				settingsDialog.dirty = true
				settingsDialog.draft.Apply()
				settingsDialog.rebuildPanel()
			})

		core.NewButton(bar).
			SetType(core.ButtonOutlined).
			SetText("Cancel").
			OnClick(func(e events.Event) {
				App.Apply()
				d.Close()
			})

		save := core.NewButton(bar).
			SetText("Save").
			SetIcon(icons.Save)
		save.OnClick(func(e events.Event) {
			if !settingsDialog.dirty {
				d.Close()
				return
			}
			*App = settingsDialog.draft
			SaveOrLog()
			settingsDialog.dirty = false
			if settingsDialog.draft.Data.WorkspacePath != settingsDialog.savedWorkspacePath {
				core.MessageDialog(d, "Workspace path changed. Restart the application to apply changes.", "Info")
			}
			d.Close()
		})
	})

	st := d.NewDialog(ctx).SetResizable(false)
	if rw := ctx.AsWidget().Scene.RenderWindow(); rw != nil {
		if main := rw.MainScene(); main != nil {
			st.SetPos(main.AsWidget().ContextMenuPos(nil))
		}
	}
	st.Run()
}

func buildSettingsBody(d *core.Body) {
	d.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Padding.Zero()
	})

	tree.AddChildAt(d, "layout", func(row *core.Frame) {
		row.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Direction = styles.Row
			s.Padding.Zero()
		})

		tree.AddChildAt(row, "sidebar", func(sidebar *core.Frame) {
			sidebar.Styler(func(s *styles.Style) {
				s.Background = colors.Scheme.SurfaceContainerLow
				s.Direction = styles.Column
				s.Min.X.Dp(160)
				s.Max.X.Dp(160)
				s.Grow.Set(0, 1)
				s.Padding.Set(units.Dp(4))
				s.Gap.Set(units.Dp(2))
				s.Overflow.Set(styles.OverflowAuto)
			})

			settingsDialog.nav = core.NewFrame(sidebar)
			settingsDialog.nav.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Gap.Set(units.Dp(2))
				s.Grow.Set(1, 0)
			})
			for _, sec := range sections {
				sec := sec
				btn := core.NewButton(settingsDialog.nav).
					SetType(core.ButtonText).
					SetText(sec.title).
					SetIcon(sec.icon)
				btn.Styler(func(s *styles.Style) {
					s.Justify.Content = styles.Start
					s.Grow.Set(1, 0)
					s.Padding.Set(units.Dp(6), units.Dp(8))
					if sec.id == settingsDialog.sectionID {
						s.Background = colors.Scheme.Secondary.Container
						s.Color = colors.Scheme.Secondary.OnContainer
					}
				})
				btn.OnClick(func(e events.Event) {
					selectSection(sec.id)
				})
			}
		})

		tree.AddChildAt(row, "content", func(content *core.Frame) {
			content.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Grow.Set(1, 1)
				s.Padding.Set(units.Dp(6), units.Dp(12))
				s.Gap.Set(units.Dp(6))
				s.Overflow.Set(styles.OverflowAuto)
			})

			settingsDialog.sectionList = core.NewFrame(content)
			settingsDialog.sectionList.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Grow.Set(1, 1)
			})
			settingsDialog.rebuildPanel()
		})
	})
}

func onPanelChange() {
	settingsDialog.dirty = true
	settingsDialog.draft.Apply()
	if settingsDialog.sectionID == "scripting" {
		settingsDialog.rebuildPanel()
	}
}

func (s *dialogState) rebuildPanel() {
	if s.sectionList == nil {
		return
	}
	s.sectionList.DeleteChildren()
	buildSettingPanel(s.sectionList, &s.draft, s.sectionID, onPanelChange)
	s.sectionList.Update()
}

func selectSection(id string) {
	if id == settingsDialog.sectionID {
		return
	}
	settingsDialog.sectionID = id
	settingsDialog.rebuildPanel()
	settingsDialog.nav.Update()
}
