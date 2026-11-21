package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ipmw "github.com/jacob-lineberry/ipecho.dev/internal/middleware"
)

// contextIP runs a request through CloudflareClientIP and returns the value the
// next handler sees under ClientIPKey. It fails the test if the key holds a
// value that is not a string.
func contextIP(t *testing.T, header http.Header) string {
	t.Helper()

	var got string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(ipmw.ClientIPKey)
		s, ok := v.(string)
		if !ok {
			t.Fatalf("context value = %#v, want a string", v)
		}
		got = s
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	ipmw.CloudflareClientIP()(next).ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestCloudflareClientIP(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   string
	}{
		{
			name:   "valid ipv4",
			header: http.Header{"Cf-Connecting-Ip": {"203.0.113.9"}},
			want:   "203.0.113.9",
		},
		{
			name:   "valid ipv6 is normalized",
			header: http.Header{"Cf-Connecting-Ip": {"2001:DB8::1"}},
			want:   "2001:db8::1",
		},
		{
			name:   "missing header",
			header: http.Header{},
			want:   "",
		},
		{
			name:   "malformed ip",
			header: http.Header{"Cf-Connecting-Ip": {"not-an-ip"}},
			want:   "",
		},
		{
			name:   "ip with port is rejected",
			header: http.Header{"Cf-Connecting-Ip": {"203.0.113.9:443"}},
			want:   "",
		},
		{
			name:   "x-forwarded-for alone is ignored",
			header: http.Header{"X-Forwarded-For": {"203.0.113.9"}},
			want:   "",
		},
		{
			name: "cloudflare header wins over x-forwarded-for",
			header: http.Header{
				"Cf-Connecting-Ip": {"203.0.113.9"},
				"X-Forwarded-For":  {"198.51.100.7"},
			},
			want: "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contextIP(t, tt.header); got != tt.want {
				t.Errorf("context IP = %q, want %q", got, tt.want)
			}
		})
	}
}
