// Package middleware provides HTTP middleware for IP extraction and proxy handling.
package middleware

import (
	"context"
	"net/http"
	"net/netip"
)

type contextKey string

const ClientIPKey contextKey = "clientIP"

// CloudflareClientIP returns middleware that stores the client IP from the
// CF-Connecting-IP header in the request context. Cloudflare sets that header,
// so the service must only be reached through Cloudflare. A missing or
// malformed header leaves the context value empty.
func CloudflareClientIP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var clientIP string

			// try Cloudflare specific header first
			if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
				if ip, err := netip.ParseAddr(cfIP); err == nil {
					clientIP = ip.String()
				}
			}

			ctx := context.WithValue(r.Context(), ClientIPKey, clientIP)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
