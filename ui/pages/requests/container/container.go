package container

import (
	"gioui.org/layout"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type Container interface {
	Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions
	DataChanged() bool
	Close() error
}
