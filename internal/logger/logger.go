package logger

func Info(message string) {
	// bus.Publish(state.LogSubmitted, domain.Log{Time: time.Now(), Level: "info", Message: message})
}

func Error(message string) {
	// bus.Publish(state.LogSubmitted, domain.Log{Time: time.Now(), Level: "error", Message: message})
}

func Warn(message string) {
	// bus.Publish(state.LogSubmitted, domain.Log{Time: time.Now(), Level: "warn", Message: message})
}
