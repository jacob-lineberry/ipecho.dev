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
		{"ipv6 context IP is reduced to its /64", "2001:db8:1:2:3:4:5:6", "2001:db8:1:2::"},
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

// Addresses inside one IPv6 /64 must share a bucket, so rotating addresses in a
// block cannot be used to dodge the limit.
func TestRateLimitKeySharesBucketWithinIPv6Prefix(t *testing.T) {
	key := func(ip string) string {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(context.WithValue(req.Context(), ipmw.ClientIPKey, ip))
		got, err := rateLimitKeyFromContext(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return got
	}

	if a, b := key("2001:db8:1:2::1"), key("2001:db8:1:2:ffff:ffff:ffff:ffff"); a != b {
		t.Errorf("same /64 got different keys: %q vs %q", a, b)
	}
	if a, b := key("2001:db8:1:2::1"), key("2001:db8:1:3::1"); a == b {
		t.Errorf("different /64s share key %q", a)
	}
}

func TestRateLimitKeyFallbackHandlesIPv6RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8:1:2::9]:51234"

	got, err := rateLimitKeyFromContext(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "2001:db8:1:2::"; got != want {
		t.Errorf("key = %q, want %q", got, want)
	}
}
