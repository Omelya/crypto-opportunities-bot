package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware додає CORS headers
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Перевірка чи origin дозволений
			if isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 години

			// Обробка preflight request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed перевіряє чи origin дозволений
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	// Якщо список порожній, дозволяємо всі (тільки для dev!)
	if len(allowedOrigins) == 0 {
		return true
	}

	for _, allowed := range allowedOrigins {
		// Wildcard support
		if allowed == "*" {
			return true
		}

		// Exact match
		if origin == allowed {
			return true
		}

		// Wildcard subdomain (*.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}
