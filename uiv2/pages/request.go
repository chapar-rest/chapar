package pages

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// RequestPage returns a builder for a request detail tab body.
func RequestPage(name string) func(content *core.Frame) {
	return func(content *core.Frame) {
		content.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Padding.Set(units.Dp(16))
			s.Gap.Set(units.Dp(8))
		})
		core.NewText(content).
			SetType(core.TextHeadlineSmall).
			SetText(name)
		core.NewText(content).
			SetType(core.TextBodyMedium).
			SetText("Request page placeholder")
	}
}
