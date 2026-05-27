package events

import "github.com/chapar-rest/chapar/internal/domain"

var (
	WorkspaceChangeTopic     = NewBroker[*domain.Workspace](defaultBufferSize)
	WorkspaceSelectedTopic   = NewBroker[*domain.Workspace](defaultBufferSize)
	EnvironmentChangeTopic   = NewBroker[*domain.Environment](defaultBufferSize)
	EnvironmentSelectedTopic = NewBroker[*domain.Environment](defaultBufferSize)
)
