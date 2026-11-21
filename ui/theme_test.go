package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/theme"
)

// Every type role must survive the UI font size scaling: the text helpers
// (ui.Overline, ui.Headline, ui.Label) read their size from the current theme.
func TestScaleTypographyKeepsEveryRole(t *testing.T) {
	applyChaparTheme("dark")
	scaleTypography(16)
	ty := theme.Current().Typography
	def := theme.DefaultTypography()
	roles := map[string][2]theme.TypographyStyle{
		"Display":   {ty.Display, def.Display},
		"Headline":  {ty.Headline, def.Headline},
		"BodySmall": {ty.BodySmall, def.BodySmall},
		"Overline":  {ty.Overline, def.Overline},
		"Label":     {ty.Label, def.Label},
	}
	for name, r := range roles {
		got, base := r[0], r[1]
		if got.Size <= base.Size || got.Weight != base.Weight {
			t.Errorf("%s = %+v, want scaled up from %+v", name, got, base)
		}
	}
	scaleTypography(defaultUIFontSize)
}
