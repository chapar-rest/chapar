package logger

import (
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
)

var Default = New()

type Logger struct {
	logs []domain.Log

	changed bool
}

func New() *Logger {
	return &Logger{
		logs: make([]domain.Log, 0),
	}
}

func (l *Logger) AddLog(log domain.Log) {
	// add log and make sure to only keep the last 1000 logs
	l.logs = append(l.logs, log)
	if len(l.logs) > 1000 {
		l.logs = l.logs[len(l.logs)-1000:]
	}
	l.changed = true
}

func GetLogs() []domain.Log {
	return Default.GetLogs()
}

func (l *Logger) GetLogs() []domain.Log {
	return l.logs
}

func Clear() {
	Default.ClearLogs()
}

func (l *Logger) ClearLogs() {
	l.logs = make([]domain.Log, 0)
	l.changed = true
}

func (l *Logger) Changed() bool {
	changed := l.changed
	l.changed = false
	return changed
}

func Info(message string) {
	Default.AddLog(domain.Log{
		Time:    time.Now(),
		Level:   "info",
		Message: message,
	})
}

func Error(message string) {
	Default.AddLog(domain.Log{
		Time:    time.Now(),
		Level:   "error",
		Message: message,
	})
}

func Warn(message string) {
	Default.AddLog(domain.Log{
		Time:    time.Now(),
		Level:   "warn",
		Message: message,
	})
}

func Print(message string) {
	Default.AddLog(domain.Log{
		Time:    time.Now(),
		Level:   "print",
		Message: message,
	})
}
