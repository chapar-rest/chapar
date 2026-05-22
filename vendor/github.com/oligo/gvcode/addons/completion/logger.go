package completion

import "log/slog"

const (
	logGroup = "completion"
)

var logger *slog.Logger

func init() {
	logger = slog.Default().WithGroup(logGroup)
}

func SetLogger(log *slog.Logger) {
	logger = log.WithGroup(logGroup)
}
