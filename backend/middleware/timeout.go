package middleware

import (
	"context"
	"net/http"
	"time"
)

// TimeoutMiddleware wraps each request with a timeout context
// Default timeout: 30 seconds for most requests
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a new request with the timeout context
			r = r.WithContext(ctx)

			// Use a response writer wrapper to detect if headers were already written
			wrapped := &timeoutResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Run the handler in a goroutine so we can timeout
			done := make(chan struct{})
			go func() {
				next.ServeHTTP(wrapped, r)
				close(done)
			}()

			select {
			case <-done:
				// Handler completed in time
				return
			case <-ctx.Done():
				// Context deadline exceeded
				if !wrapped.headerWritten {
					w.WriteHeader(http.StatusRequestTimeout)
					w.Write([]byte("Request timeout"))
				}
				return
			}
		})
	}
}

// timeoutResponseWriter wraps http.ResponseWriter to track if headers were written
type timeoutResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (w *timeoutResponseWriter) WriteHeader(statusCode int) {
	w.headerWritten = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *timeoutResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.headerWritten = true
	}
	return w.ResponseWriter.Write(b)
}
