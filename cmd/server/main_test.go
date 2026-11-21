package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ipmw "github.com/jacob-lineberry/ipecho.dev/internal/middleware"
)

func TestRateLimitKeyFromContext(t *testing.T) {
	const remoteAddr = "198.51.100.7:51234"

	tests := []struct {
		name  string
		value any // stored under ClientIPKey; nil means the key is not set
		want  string
	}{
		{"uses the context IP", "203.0.113.9", "203.0.113.9"},
		{"empty context IP falls back to the connection address", "", "198.51.100.7"},
		{"missing context IP falls back to the connection address", nil, "198.51.100.7"},
		{"non-string context value falls back to the connection address", 42, "198.51.100.7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = remoteAddr
			if tt.value != nil {
				req = req.WithContext(context.WithValue(req.Context(), ipmw.ClientIPKey, tt.value))
			}

			got, err := rateLimitKeyFromContext(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("key = %q, want %q", got, tt.want)
			}
		})
	}
}
