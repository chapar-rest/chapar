package state

import (
	"github.com/mirzakhany/chapar/internal/domain"
)

var GlobalState = State{
	SelectedEnv: nil,
}

type State struct {
	SelectedEnv *domain.Environment
}
