// Package logger builds the application's structured JSON logger.
package logger

import (
	"io"
	"log/slog"
)

// New returns a JSON slog logger that writes to w at the given minimum level.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}
