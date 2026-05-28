package pages

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/domain"
)

// EnvironmentPage returns a builder for an environment detail tab body.
func EnvironmentPage(env *domain.Environment) func(content *core.Frame) {
	name := env.GetName()
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
			SetText(fmt.Sprintf("Environment page placeholder (%d variables)", len(env.Spec.Values)))
	}
}
