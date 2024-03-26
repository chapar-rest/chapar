package state

import "errors"

type Action string

const (
	ActionAdd    Action = "add"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

var ErrNotFound = errors.New("ErrNotFound")
