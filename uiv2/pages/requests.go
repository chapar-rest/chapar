package pages

import (
	"github.com/mirzakhany/yoga/ui"
)

type Requests struct {
	tree *ui.Tree
}

func NewRequestsPage(tree *ui.Tree) *Requests {
	return &Requests{tree: tree}
}
