package uiv2

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const defaultUIFontSize = 14

func applyChaparTheme(name string) {
	if name == "" || !theme.Use(name) {
		theme.Use("yoga-dark")
	}
}

func applyChaparAppearance(general domain.GeneralConfig, editor domain.EditorConfig) {
	applyChaparTheme(general.Theme)
	applyChaparFonts(general, editor)
}

func applyChaparFonts(general domain.GeneralConfig, editor domain.EditorConfig) {
	uiSize := general.UIFontSize
	if uiSize <= 0 {
		uiSize = defaultUIFontSize
	}
	editorSize := editor.FontSize
	if editorSize <= 0 {
		editorSize = 12
	}
	tabWidth := editor.TabWidth
	if tabWidth <= 0 {
		tabWidth = 4
	}

	_ = yoga.SetFont(shape.FontConfig{
		UI: shape.FaceConfig{Size: float32(uiSize)},
		Mono: shape.FaceConfig{
			Size:   float32(editorSize),
			Family: editor.FontFamily,
		},
		TabWidth: tabWidth,
	})

	scaleTypography(uiSize)
}

func scaleTypography(bodySize int) {
	if bodySize <= 0 {
		bodySize = defaultUIFontSize
	}
	factor := float32(bodySize) / defaultUIFontSize
	def := theme.DefaultTypography()
	th := theme.Current()
	th.Typography = theme.Typography{
		Caption:    scaleTypographyStyle(def.Caption, factor),
		Body:       scaleTypographyStyle(def.Body, factor),
		BodyStrong: scaleTypographyStyle(def.BodyStrong, factor),
		Subtitle:   scaleTypographyStyle(def.Subtitle, factor),
		Title:      scaleTypographyStyle(def.Title, factor),
	}
}

func scaleTypographyStyle(s theme.TypographyStyle, factor float32) theme.TypographyStyle {
	return theme.TypographyStyle{
		Size:       render.Px(float32(s.Size) * factor),
		LineHeight: render.Px(float32(s.LineHeight) * factor),
		Weight:     s.Weight,
	}
}
