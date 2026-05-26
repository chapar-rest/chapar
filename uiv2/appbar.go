package uiv2

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/uiv2/settings"
)

const appBarHeight = 50

func NewAppBar(b *core.Body) {
	b.AddTopBar(func(bar *core.Frame) {
		bar.Styler(func(s *styles.Style) {
			s.Min.Y.Dp(appBarHeight)
			s.Grow.Set(1, 0)
			s.Padding.SetHorizontal(units.Dp(4))
			s.Gap.Set(units.Dp(4))
		})

		core.NewText(bar).
			SetText("Chapar").
			SetType(core.TextTitleMedium)

		center := core.NewFrame(bar)
		center.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Justify.Content = styles.Center
		})
		search := core.NewTextField(center).
			SetPlaceholder("Search...").
			SetLeadingIcon(icons.Search)
		search.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Max.X.Ch(40)
		})

		core.NewChooser(bar).SetItems(
			core.ChooserItem{Value: "default", Text: "Default"},
			core.ChooserItem{Value: "dev", Text: "Development"},
			core.ChooserItem{Value: "prod", Text: "Production"},
		)

		settingsBtn := core.NewButton(bar).
			SetIcon(icons.Settings).
			SetTooltip("Settings")
		settingsBtn.OnClick(func(e events.Event) {
			settings.OpenDialog(settingsBtn)
		})
	})
}
