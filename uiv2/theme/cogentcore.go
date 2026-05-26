package theme

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

func applyToCogentCore(primary color.RGBA, isDark bool) {
	core.AppColor = primary
	colors.SetSchemes(primary)
	colors.SetScheme(isDark)

	if isDark {
		core.AppearanceSettings.Theme = core.ThemeDark
	} else {
		core.AppearanceSettings.Theme = core.ThemeLight
	}
	core.AppearanceSettings.Color = primary

	core.AppearanceSettings.Text.SansSerif = defaultSansFont
	core.AppearanceSettings.Text.Monospace = defaultMonoFont
	rich.Settings = core.AppearanceSettings.Text

	core.UpdateAll()
}

func setupGlobalStyles() {
	core.TheApp.SetSceneInit(func(sc *core.Scene) {
		sc.SetWidgetInit(func(w core.Widget) {
			wb := w.AsWidget()
			wb.Styler(func(s *styles.Style) {
				s.Font.Family = rich.SansSerif
			})

			switch w := w.(type) {
			case *core.TextField:
				w.SetType(core.TextFieldOutlined)
				w.FinalStyler(func(s *styles.Style) {
					s.Border.Radius = styles.BorderRadiusSmall
					s.Padding.Set(units.Dp(10), units.Dp(8))
				})
			case *core.Button:
				w.FinalStyler(func(s *styles.Style) {
					s.Border.Radius = styles.BorderRadiusSmall
				})
			case *core.Chooser:
				w.FinalStyler(func(s *styles.Style) {
					s.Border.Radius = styles.BorderRadiusSmall
				})
			}
		})
	})
}
