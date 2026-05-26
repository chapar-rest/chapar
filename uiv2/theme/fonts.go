package theme

import (
	"embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
)

//go:embed fonts/*
var embeddedFonts embed.FS

const (
	defaultSansFont = "Source Sans Pro"
	defaultMonoFont = "JetBrains Mono"
)

func setupFonts() {
	fonts.AddEmbedded(embeddedFonts)

	core.AppearanceSettings.Text.SansSerif = defaultSansFont
	core.AppearanceSettings.Text.Monospace = defaultMonoFont
	rich.Settings = core.AppearanceSettings.Text
}
