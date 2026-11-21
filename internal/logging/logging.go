// Package logging configures structured logging for Cloud Run.
package logging

import (
	"io"
	"log/slog"
)

// New returns a logger that writes one JSON object per line to w. Cloud Logging
// parses each line into jsonPayload and reads the "severity" and "message"
// fields, so slog's "level" and "msg" keys are renamed to match.
func New(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: replaceAttr,
	}))
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		if level, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(severity(level))
		}
	case slog.MessageKey:
		a.Key = "message"
	}
	return a
}

// severity maps a slog level to the names Cloud Logging accepts.
func severity(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARNING"
	default:
		return "ERROR"
	}
}
