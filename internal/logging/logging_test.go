package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestSeverity(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelInfo + 2, "INFO"},
		{slog.LevelWarn, "WARNING"},
		{slog.LevelError - 1, "WARNING"},
		{slog.LevelError, "ERROR"},
		{slog.LevelError + 4, "ERROR"},
	}
	for _, tt := range tests {
		if got := severity(tt.level); got != tt.want {
			t.Errorf("severity(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestNewWritesCloudLoggingFields(t *testing.T) {
	tests := []struct {
		name string
		log  func(l *slog.Logger)
		want string
	}{
		{"info", func(l *slog.Logger) { l.Info("m") }, "INFO"},
		{"warn is WARNING", func(l *slog.Logger) { l.Warn("m") }, "WARNING"},
		{"error", func(l *slog.Logger) { l.Error("m") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(New(&buf))

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("output %q is not one JSON object: %v", buf.String(), err)
			}
			if got := entry["severity"]; got != tt.want {
				t.Errorf("severity = %v, want %q", got, tt.want)
			}
			if got := entry["message"]; got != "m" {
				t.Errorf("message = %v, want %q", got, "m")
			}
			for _, key := range []string{"level", "msg"} {
				if _, ok := entry[key]; ok {
					t.Errorf("entry still has the %q key: %v", key, entry)
				}
			}
		})
	}
}

func TestNewKeepsAttributesOnOneLine(t *testing.T) {
	var buf bytes.Buffer
	New(&buf).Info("starting", "version", "1.2.3", "addr", ":8080")

	if !strings.HasSuffix(buf.String(), "\n") || strings.Count(buf.String(), "\n") != 1 {
		t.Errorf("want exactly one newline-terminated line, got %q", buf.String())
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if entry["version"] != "1.2.3" || entry["addr"] != ":8080" {
		t.Errorf("attributes missing from %v", entry)
	}
}
