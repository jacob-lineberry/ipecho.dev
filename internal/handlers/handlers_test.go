package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jacob-lineberry/ipecho.dev/internal/handlers"
	ipmw "github.com/jacob-lineberry/ipecho.dev/internal/middleware"
)

// request builds a GET request. A non-empty ctxIP is stored under ClientIPKey
// the way the middleware does it; a non-empty remoteAddr replaces the default.
func request(ctxIP, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	if ctxIP != "" {
		req = req.WithContext(context.WithValue(req.Context(), ipmw.ClientIPKey, ctxIP))
	}
	return req
}

func assertHeaders(t *testing.T, rec *httptest.ResponseRecorder, contentType string) {
	t.Helper()
	if got := rec.Code; got != http.StatusOK {
		t.Errorf("status = %d, want %d", got, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != contentType {
		t.Errorf("Content-Type = %q, want %q", got, contentType)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}
}

func TestPlainIP(t *testing.T) {
	rec := httptest.NewRecorder()
	handlers.PlainIP(rec, request("203.0.113.9", ""))

	assertHeaders(t, rec, "text/plain; charset=utf-8")
	if got, want := rec.Body.String(), "203.0.113.9\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestJSONIP(t *testing.T) {
	rec := httptest.NewRecorder()
	handlers.JSONIP(rec, request("2001:db8::1", ""))

	assertHeaders(t, rec, "application/json; charset=utf-8")
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not valid JSON: %v", rec.Body.String(), err)
	}
	if len(body) != 1 || body["ip"] != "2001:db8::1" {
		t.Errorf("body = %v, want only {\"ip\": \"2001:db8::1\"}", body)
	}
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	handlers.Health(rec, request("", ""))

	assertHeaders(t, rec, "text/plain; charset=utf-8")
	if got, want := rec.Body.String(), "ok\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// Without a context IP the handlers fall back to the connection's address.
func TestClientIPFallsBackToRemoteAddr(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{"ipv4 with port", "198.51.100.7:51234", "198.51.100.7\n"},
		{"ipv6 with port", "[2001:db8::7]:51234", "2001:db8::7\n"},
		{"ipv4 without port", "198.51.100.7", "198.51.100.7\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handlers.PlainIP(rec, request("", tt.remoteAddr))
			if got := rec.Body.String(); got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

// The context IP takes priority over the connection's address.
func TestContextIPWinsOverRemoteAddr(t *testing.T) {
	rec := httptest.NewRecorder()
	handlers.PlainIP(rec, request("203.0.113.9", "198.51.100.7:51234"))

	if got, want := rec.Body.String(), "203.0.113.9\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}
