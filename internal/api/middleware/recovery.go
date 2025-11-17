package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware ловить паніки та повертає 500 помилку
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Логуємо panic з stack trace
				log.Printf("❌ PANIC: %v\n%s", err, debug.Stack())

				// Відправляємо 500 помилку
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"Internal server error","message":"%v"}`, err)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
