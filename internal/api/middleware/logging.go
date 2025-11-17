package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter обгортка для захоплення status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// LoggingMiddleware логує всі HTTP запити
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Обгортка для response writer
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Викликати наступний handler
		next.ServeHTTP(wrapped, r)

		// Логування
		duration := time.Since(start)
		log.Printf(
			"📡 %s %s | Status: %d | Duration: %v | Size: %d bytes | IP: %s",
			r.Method,
			r.RequestURI,
			wrapped.statusCode,
			duration,
			wrapped.written,
			r.RemoteAddr,
		)
	})
}
