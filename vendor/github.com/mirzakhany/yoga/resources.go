package yoga

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/ui"
)

// Package-level resource singletons.
//
// The yoga runtime populates these when yoga.New() succeeds (GPU build) or when
// the caller invokes SetResources (headless / test builds). Components call
// Text(), Icons(), and Clipboard() instead of receiving dependencies through
// constructors, so no threading of runtime objects through the UI tree is
// needed.
var (
	globalText  *shape.Engine
	globalIcons *render.SpriteSheet
	globalClip  input.Clipboard
)

// Text returns the application's shared text-shaping engine.
// Valid after yoga.New() returns (GPU build) or after SetResources is called.
func Text() *shape.Engine { return globalText }

// SetFont reconfigures the application's fonts, sizes, letter spacing, line
// height, and editor tab width at runtime. Call it any time after yoga.New()
// (GPU build) or SetResources. Editors refresh on the next frame.
func SetFont(cfg shape.FontConfig) error { return globalText.SetFont(cfg) }

// Icons returns the application's shared icon sprite sheet.
// Valid after yoga.New() returns (GPU build) or after SetResources is called.
func Icons() *render.SpriteSheet { return globalIcons }

// Clipboard returns the platform clipboard (GLFW in GPU builds, in-memory in
// headless/test builds).
// Valid after yoga.New() returns (GPU build) or after SetResources is called.
func Clipboard() input.Clipboard { return globalClip }

// SetResources registers the runtime resources used by all components.
// The GPU App calls this automatically during yoga.New(); call it manually in
// headless binaries and test helpers before constructing any components.
//
//	text, _ := shape.NewEngine(1, false)
//	sheet  := render.NewSpriteSheet(text.Atlas)
//	yoga.SetResources(text, sheet, &input.MemClipboard{})
func SetResources(text *shape.Engine, icons *render.SpriteSheet, clip input.Clipboard) {
	globalText = text
	globalIcons = icons
	globalClip = clip
	ui.SetFrameResources(text, icons, clip)
}
