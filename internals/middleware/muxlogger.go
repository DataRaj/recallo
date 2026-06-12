package middleware

import (
	"net/http"
	"time"

	"recallo/internals/logger"
)

// responseWriter wraps http.ResponseWriter to capture the status code
// written by downstream handlers so the logger middleware can report it.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Loggingmiddleware logs every inbound request together with the HTTP method,
// path, remote address, response status, and elapsed time. Output goes through
// the shared logger so lines appear in both stdout and logs/app.log.
func Loggingmiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		logger.App.Printf("[REQUEST]  method=%s path=%s remote=%s",
			r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(rw, r)

		elapsed := time.Since(start)
		logger.App.Printf("[RESPONSE] method=%s path=%s status=%d latency=%s",
			r.Method, r.URL.Path, rw.status, elapsed)
	})
}
