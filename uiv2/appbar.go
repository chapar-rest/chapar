package uiv2

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func NewAppBar(b *core.Body) {
	b.AddTopBar(func(bar *core.Frame) {
		bar.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Padding.Set(units.Dp(4), units.Dp(8))
			s.Gap.Set(units.Dp(2))
			s.Justify.Content = styles.SpaceAround
			s.Align.Items = styles.Center
			s.Border.Style.Bottom = styles.BorderSolid
			s.Border.Width.Bottom = units.Dp(1)
			s.Border.Color.Bottom = colors.Scheme.OutlineVariant
		})

		left := core.NewFrame(bar)
		left.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Justify.Content = styles.Start
		})

		core.NewText(left).
			SetText("Chapar").
			SetType(core.TextTitleMedium).
			Styler(func(s *styles.Style) {
				s.Padding.Set(units.Dp(4), units.Dp(12))
				s.Min.X.Dp(90)
				s.Max.X.Dp(90)
			})

		workspaceChooser := core.NewChooser(left).SetItems(
			core.ChooserItem{Value: "default", Text: "Default Workspace"},
			core.ChooserItem{Value: "personal", Text: "Personal"},
			core.ChooserItem{Value: "work", Text: "Work"},
		)
		workspaceChooser.Styler(func(s *styles.Style) {
			s.Min.X.Dp(160)
			s.Max.X.Dp(160)
		})

		right := core.NewFrame(bar)
		right.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Justify.Content = styles.End
		})

		chooser := core.NewChooser(right).SetItems(
			core.ChooserItem{Value: "default", Text: "Default"},
			core.ChooserItem{Value: "dev", Text: "Development"},
			core.ChooserItem{Value: "prod", Text: "Production"},
		)
		chooser.Styler(func(s *styles.Style) {
			s.Min.X.Dp(160)
			s.Max.X.Dp(160)
		})

		// settingsBtn := core.NewButton(right).
		// 	SetType(core.ButtonAction).
		// 	SetIcon(icons.Settings).
		// 	SetTooltip("Settings")
		// settingsBtn.Styler(func(s *styles.Style) {
		// 	s.Padding.Set(units.Dp(6), units.Dp(6))
		// })
		// settingsBtn.OnClick(func(e events.Event) {
		// 	settings.OpenDialog(settingsBtn)
		// })
	})
}
