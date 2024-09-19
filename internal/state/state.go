package state

import "errors"

type (
	Action string
	Source string
)

const (
	ActionAdd    Action = "add"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"

	SourceView        Source = "view"
	SourceFile        Source = "file"
	SourceController  Source = "controller"
	SourceRestService Source = "rest-service"
	SourceGRPCService Source = "grpc-service"
)

var ErrNotFound = errors.New("ErrNotFound")
