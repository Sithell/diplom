package logger

import (
	"log"
	"os"

	"github.com/maarulav/k8s-setup/internal/status"
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
	status *status.SetupStatus
}

// New creates a new Logger instance
func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// SetStatus sets the current setup status
func (l *Logger) SetStatus(status *status.SetupStatus) {
	l.status = status
}

// GetStatus returns the current setup status
func (l *Logger) GetStatus() *status.SetupStatus {
	return l.status
}
