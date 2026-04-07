package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// SelectiveTimeoutMiddleware applies timeout only to non-WebSocket routes
// WebSocket connections are long-lived and should not be subject to request timeouts
func SelectiveTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip timeout for WebSocket routes (long-lived connections)
			if strings.HasPrefix(r.URL.Path, "/ws") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip timeout for health check endpoints
			if strings.HasPrefix(r.URL.Path, "/health") || strings.HasPrefix(r.URL.Path, "/readiness") {
				next.ServeHTTP(w, r)
				return
			}

			// Apply timeout to API routes
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			wrapped := &timeoutResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Run handler in goroutine so we can timeout
			done := make(chan struct{})
			go func() {
				next.ServeHTTP(wrapped, r)
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				if !wrapped.headerWritten {
					w.WriteHeader(http.StatusRequestTimeout)
					w.Write([]byte("Request timeout"))
				}
				return
			}
		})
	}
}
