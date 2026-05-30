package theme

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

func applyToCogentCore(def schema) {
	core.AppColor = def.primary
	colors.SetSchemes(def.primary)
	colors.SetScheme(def.isDark)

	if def.hasExplicitPalette() {
		applyExplicitPalette(def)
	}

	if def.isDark {
		core.AppearanceSettings.Theme = core.ThemeDark
	} else {
		core.AppearanceSettings.Theme = core.ThemeLight
	}
	core.AppearanceSettings.Color = def.primary

	core.AppearanceSettings.Text.SansSerif = defaultSansFont
	core.AppearanceSettings.Text.Monospace = defaultMonoFont
	rich.Settings = core.AppearanceSettings.Text

	core.UpdateAll()
}

func applyExplicitPalette(def schema) {
	sc := colors.Scheme
	sc.Surface = colors.Uniform(def.surface)
	sc.SurfaceContainerLowest = colors.Uniform(def.surfaceContainerLowest)
	sc.SurfaceContainerLow = colors.Uniform(def.surfaceContainerLow)
	sc.SurfaceContainer = colors.Uniform(def.surfaceContainer)
	sc.SurfaceContainerHigh = colors.Uniform(def.surfaceContainerHigh)
	sc.SurfaceContainerHighest = colors.Uniform(def.surfaceContainerHighest)
	sc.SurfaceVariant = colors.Uniform(def.surfaceVariant)
	sc.OnSurface = colors.Uniform(def.onSurface)
	sc.OnSurfaceVariant = colors.Uniform(def.onSurfaceVariant)
	sc.Outline = colors.Uniform(def.outline)
	sc.OutlineVariant = colors.Uniform(def.outlineVariant)
	sc.Primary.Base = colors.Uniform(def.primary)
	sc.Primary.On = colors.Uniform(def.onPrimary)
	sc.Secondary.Base = colors.Uniform(def.secondary)
	sc.Secondary.On = colors.Uniform(def.onSecondary)
	sc.Secondary.Container = colors.Uniform(def.secondaryContainer)
	sc.Secondary.OnContainer = colors.Uniform(def.onSecondaryContainer)
}

func setupGlobalStyles() {
	core.TheApp.SetSceneInit(func(sc *core.Scene) {
		sc.Styler(func(s *styles.Style) {
			s.Padding.Zero()
			s.Gap.Zero()
		})
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
					s.Padding.Set(units.Dp(4), units.Dp(8))
					s.Min.Y.Dp(22)
					s.Max.Y.Dp(26)
					s.Font.Size.Dp(14)
					s.Text.LineHeight = 1.25
					s.Grow.Set(1, 0)
					s.Min.X.Ch(8)
					s.Max.X.Ch(0) // drop default 40ch cap so fields can fill containers
					if !s.Is(states.Focused) {
						s.Border.Style.Set(styles.BorderSolid)
						s.Border.Width.Set(units.Dp(1))
						s.Border.Color.Set(colors.Scheme.OutlineVariant)
					}
				})
			case *core.Button:
				w.FinalStyler(func(s *styles.Style) {
					s.Border.Radius = styles.BorderRadiusSmall
					s.Padding.Set(units.Dp(4), units.Dp(8))
					s.Font.Size.Dp(13)
					s.Min.Y.Dp(16)
					s.Gap.Set(units.Dp(4))
				})
			case *core.Switch:
				w.FinalStyler(func(s *styles.Style) {
					s.Padding.SetVertical(units.Dp(2))
					s.Padding.SetHorizontal(units.Dp(4))
				})
			case *core.Table:
				w.TableStyler = func(w core.Widget, s *styles.Style, row, col int) {
					s.Gap.Set(units.Dp(2))
					s.Border.Radius.Set(units.Dp(2))
				}
				w.FinalStyler(func(s *styles.Style) {
					s.Gap.Set(units.Dp(2))
					s.Border.Radius.Set(units.Dp(2))
				})
			case *core.Chooser:
				w.SetType(core.ChooserOutlined).SetIndicator(icons.ExpandMore)
				tree.AddChildInit(w, "text", func(w *core.Text) {
					w.Styler(func(s *styles.Style) {
						s.Grow.Set(1, 0)
						s.Text.Align = text.Start
					})
				})
				w.FinalStyler(func(s *styles.Style) {
					s.Background = nil
					s.Border.Radius = styles.BorderRadiusSmall
					s.Justify.Content = styles.Start
					s.Align.Items = styles.Center
					s.Text.Align = text.Start
					s.Text.AlignV = text.Center
					s.Padding.Set(units.Dp(4), units.Dp(8))
					s.Min.Y.Dp(18)
					s.Max.Y.Dp(22)
					if !s.Is(states.Focused) {
						s.Border.Style.Set(styles.BorderSolid)
						s.Border.Width.Set(units.Dp(1))
						s.Border.Color.Set(colors.Scheme.OutlineVariant)
					}
				})
			}
		})
	})
}
