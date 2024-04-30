package logger

import (
	"fmt"
	"log/slog"
	"os"
)


// Logger extens slog.Logger to allow setting the log level
type Logger struct {
	*slog.Logger
	level *slog.LevelVar
}


// NewLogger creates a new Logger
func NewLogger() *Logger {
	level := new(slog.LevelVar)
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	return &Logger{
		Logger: log,
		level:  level, 
	}
}


func (l *Logger) Log() *slog.Logger {
	return l.Logger
}

// SetLevel sets the loggers log level
func (l *Logger) SetLevel(level slog.Level) {
	l.level.Set(level)
}

// ParseLevel sets the level from a string
func (l *Logger) ParseLevel(levelString string) error {
	var level slog.Level
	err := level.UnmarshalText([]byte(levelString))
	if err != nil {
		return fmt.Errorf("parsing level from string: %w", err)
	}

	l.level.Set(level)
	return nil
}

