package domain

import "time"

type Log struct {
	Time    time.Time
	Level   string
	Message string
}
