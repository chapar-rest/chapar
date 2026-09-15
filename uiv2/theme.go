package uiv2

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const defaultUIFontSize = 14

func init() {
	theme.Register(chaparDark())
}

// chaparDark is Chapar's classic Gio dark palette, ported to Yoga tokens.
// Source: ui/chapartheme Switch("dark").
func chaparDark() theme.Theme {
	fg := rgb(0xd7, 0xda, 0xde)
	muted := rgb(0x8b, 0x8e, 0x95)
	accent := rgb(0x45, 0x89, 0xf5)
	success := rgb(0x8b, 0xc3, 0x4a)
	warning := rgb(0xff, 0xe0, 0x73)
	t := theme.Theme{
		Name: "dark",
		Dark: true,

		Surface:            rgb(0x20, 0x22, 0x24), // page / workspace
		Chrome:             rgb(0x2b, 0x2d, 0x31), // sidebar / tree
		ChromeMuted:        rgb(0x1a, 0x1c, 0x1e), // icon nav rail
		Foreground:         fg,
		ForegroundMuted:    muted,
		ForegroundSubtle:   rgb(0x6c, 0x6f, 0x76),
		ForegroundDisabled: rgba(0x8b, 0x8e, 0x95, 0.45),
		Accent:             accent,
		AccentHover:        rgb(0x5e, 0x9b, 0xfa),
		AccentPressed:      rgb(0x36, 0x72, 0xd8),
		AccentForeground:   rgb(0xff, 0xff, 0xff),
		Border:             rgb(0x6c, 0x6f, 0x76),
		BorderStrong:       rgb(0x8b, 0x8e, 0x95),
		ListHover:          rgb(0x35, 0x37, 0x3c),
		ListActive:         rgb(0x3a, 0x3c, 0x42),
		FocusRing:          accent,
		Selection:          rgb(0x63, 0x80, 0xad),
		ScrollTrack:        rgb(0x25, 0x27, 0x2a),
		ScrollThumb:        rgb(0x6c, 0x6f, 0x76),
		ScrollThumbHover:   accent,
		Error:              rgb(0xff, 0x73, 0x73),
		Warning:            warning,
		Success:            success,

		Spacing:    theme.DefaultSpacing(),
		Radius:     theme.DefaultRadius(),
		Stroke:     theme.DefaultStroke(),
		Typography: theme.DefaultTypography(),
		Metrics:    theme.DefaultComponentMetrics(),
		Elevation:  theme.DefaultElevationDark(),

		Syntax: map[highlight.ColorClass]render.Color{
			highlight.ClassDefault: fg,
			highlight.ClassKeyword: accent,
			highlight.ClassString:  success,
			highlight.ClassComment: muted,
			highlight.ClassNumber:  warning,
			highlight.ClassType:    rgb(0xb0, 0xb3, 0xb8),
		},
	}
	return t
}

func rgb(r, g, b uint8) render.Color {
	return render.RGBA8(r, g, b, 255)
}

func rgba(r, g, b uint8, a float32) render.Color {
	c := render.RGBA8(r, g, b, 255)
	c.A = a
	return c
}

func applyChaparTheme(name string) {
	if name == "" || !theme.Use(name) {
		theme.Use("dark")
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
