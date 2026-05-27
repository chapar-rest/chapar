package uiv2

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"

	"github.com/chapar-rest/chapar/assets"
)

var welcomeMessages = []string{
	"Welcome to Chapar",
	"Double Click on any item to open it.",
	"Use Cmd/Ctrl+s to save the changes",
	"Import your data from other apps using import functionality",
	"Using the environment dropdown you can switch between different environments",
	"Use the sidebar to navigate between different sections",
}

// NewWelcome creates the startup tips panel shown before any section or tab is open.
func NewWelcome(parent tree.Node) *core.Frame {
	frame := core.NewFrame(parent)
	frame.SetName("welcome")
	frame.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.CenterAll()
		s.Grow.Set(1, 1)
		s.Gap.Set(units.Dp(8))
		s.Padding.Set(units.Dp(24))
	})

	img := core.NewImage(frame)
	img.SetImage(assets.MustLoadImage("chapar.png"))
	img.Styler(func(s *styles.Style) {
		s.Max.X.Dp(120)
		s.Max.Y.Dp(120)
		s.Align.Self = styles.Center
	})

	for i, msg := range welcomeMessages {
		txt := core.NewText(frame).SetText(msg)
		if i == 0 {
			txt.SetType(core.TextHeadlineSmall)
		} else {
			txt.SetType(core.TextBodyMedium)
		}
		txt.Styler(func(s *styles.Style) {
			s.Align.Self = styles.Center
		})
	}
	return frame
}
