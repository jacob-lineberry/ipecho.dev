// Command server runs the ipecho.dev HTTP service on Cloud Run.
// It returns the client's IP address via simple REST endpoints.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/jacob-lineberry/ipecho.dev/internal/handlers"
	"github.com/jacob-lineberry/ipecho.dev/internal/logging"
	ipmw "github.com/jacob-lineberry/ipecho.dev/internal/middleware"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// structured JSON logs on stdout, in the fields Cloud Logging reads
	slog.SetDefault(logging.New(os.Stdout))

	// GCR provides PORT environment variable
	// Default to 8080 for local development
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// initialize chi router
	r := chi.NewRouter()

	// middleware stack, in order:
	// CloudflareClientIP -> Recoverer -> CleanPath -> Timeout
	//
	// there is no per-request logger on purpose: Cloud Run's own request log
	// already records the method, URL, status and latency of every request, and
	// this service does not log client IP addresses.

	// CloudflareClientIP: read the client IP from CF-Connecting-IP
	// must come before the rate limiter
	r.Use(ipmw.CloudflareClientIP())

	// Recoverer: recovers from panics, returns HTTP 500
	r.Use(chimw.Recoverer)

	// CleanPath: cleans double slashes from URL paths
	r.Use(chimw.CleanPath)

	// Timeout: set context deadline for requests
	// keep short as requests should fail fast
	r.Use(chimw.Timeout(10 * time.Second))

	// Health endpoint outside rate limiter
	r.Get("/health", handlers.Health)

	// rate-limited routes group
	r.Group(func(r chi.Router) {
		// rate limit: 120 requests per minute per client
		r.Use(httprate.Limit(
			120,
			1*time.Minute,
			httprate.WithKeyFuncs(rateLimitKeyFromContext),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			}),
		))

		// public endpoints
		r.Get("/", handlers.PlainIP)
		r.Get("/json", handlers.JSONIP)
	})

	// HTTP server and timeouts
	srv := &http.Server{
		Addr:    addr,
		Handler: r,

		// timeouts
		ReadHeaderTimeout: 5 * time.Second,   // time to read request headers
		ReadTimeout:       10 * time.Second,  // time to read entire request
		WriteTimeout:      10 * time.Second,  // time to write response
		IdleTimeout:       120 * time.Second, // time to keep connection alive
	}

	// start server in background goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "version", version, "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	// graceful shutdown handler
	// GCR sends SIGTERM before stopping container
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// wait for either shutdown signal or server error
	select {
	case sig := <-sigCh:
		slog.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		// server exited unexpectedly
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
		return
	}

	// graceful shutdown: allow up to 10 seconds for in-flight requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	} else {
		slog.Info("server stopped")
	}
}

// RateLimitKeyFromContext extracts the client from request context
// for use as the rate limiting key. falls back to httprate.KeyByIP
// if context value is missing.
func rateLimitKeyFromContext(r *http.Request) (string, error) {
	// try to get IP from context set by CloudflareClientIP middleware
	if v := r.Context().Value(ipmw.ClientIPKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s, nil
		}
	}

	// fallback: use httprate's default IP extraction
	// (should rarely happen if middleware is configured correctly)
	return httprate.KeyByIP(r)
}
