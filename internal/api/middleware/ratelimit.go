package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// RateLimiter представляє rate limiter
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
}

// visitor представляє одного відвідувача
type visitor struct {
	limiter  *tokenBucket
	lastSeen time.Time
}

// tokenBucket реалізація token bucket algorithm
type tokenBucket struct {
	tokens    int
	maxTokens int
	refillAt  time.Time
	mu        sync.Mutex
}

// NewRateLimiter створює новий rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     requestsPerMinute,
		window:   time.Minute,
	}

	// Запустити cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// RateLimitMiddleware middleware для rate limiting
func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Отримати IP адресу
		ip := getIP(r)

		// Перевірити rate limit
		if !rl.allow(ip) {
			respondRateLimited(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow перевіряє чи дозволений запит
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		// Створити нового visitor
		v = &visitor{
			limiter: &tokenBucket{
				tokens:    rl.rate,
				maxTokens: rl.rate,
				refillAt:  time.Now().Add(rl.window),
			},
			lastSeen: time.Now(),
		}
		rl.visitors[ip] = v
	}

	// Оновити last seen
	v.lastSeen = time.Now()

	// Перевірити чи є токени
	return v.limiter.take()
}

// take намагається взяти токен
func (tb *tokenBucket) take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Перевірити чи треба refill
	now := time.Now()
	if now.After(tb.refillAt) {
		tb.tokens = tb.maxTokens
		tb.refillAt = now.Add(time.Minute)
	}

	// Якщо є токени, взяти один
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// cleanupVisitors очищає старих visitors
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			// Видалити visitors які не робили запитів останні 10 хвилин
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getIP отримує IP адресу з запиту
func getIP(r *http.Request) string {
	// Спробувати отримати з X-Forwarded-For (якщо за proxy)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// Спробувати X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Використати RemoteAddr
	return r.RemoteAddr
}

// respondRateLimited відправляє 429 помилку
func respondRateLimited(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Rate limit exceeded",
		"message": "Too many requests. Please try again later.",
	})
}
