package uiv2

import (
	"cogentcore.org/core/core"

	"github.com/chapar-rest/chapar/uiv2/settings"
	"github.com/chapar-rest/chapar/uiv2/theme"
)

func Run() {
	theme.Init()
	settings.Init()
	settings.LoadOrLog()

	b := core.NewBody("Chapar")
	NewAppBar(b)
	b.NewWindow().SetDisplayTitle(false).RunMain()
}
