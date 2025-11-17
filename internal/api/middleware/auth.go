package middleware

import (
	"context"
	"crypto-opportunities-bot/internal/api/auth"
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"net/http"
	"strings"
)

// ContextKey тип для ключів контексту
type ContextKey string

const (
	// UserContextKey ключ для збереження даних користувача в контексті
	UserContextKey ContextKey = "admin_user"
)

// JWTAuthMiddleware перевіряє JWT токен
func JWTAuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Отримати Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "Missing authorization header")
				return
			}

			// Витягти токен
			tokenString, err := auth.ExtractTokenFromBearer(authHeader)
			if err != nil {
				respondUnauthorized(w, "Invalid authorization header format")
				return
			}

			// Валідувати токен
			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				respondUnauthorized(w, "Invalid or expired token")
				return
			}

			// Додати claims в контекст
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware що вимагає певну роль
func RequireRole(minRole models.AdminRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Отримати claims з контексту
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				respondForbidden(w, "Access denied")
				return
			}

			// Перевірити роль
			if !hasRequiredRole(claims.Role, minRole) {
				respondForbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin middleware для admin та super_admin
func RequireAdmin(next http.Handler) http.Handler {
	return RequireRole(models.AdminRoleAdmin)(next)
}

// RequireSuperAdmin middleware тільки для super_admin
func RequireSuperAdmin(next http.Handler) http.Handler {
	return RequireRole(models.AdminRoleSuperAdmin)(next)
}

// GetUserFromContext отримує user claims з контексту
func GetUserFromContext(ctx context.Context) *auth.Claims {
	claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// hasRequiredRole перевіряє чи користувач має достатньо прав
func hasRequiredRole(userRole, requiredRole models.AdminRole) bool {
	// Ієрархія ролей: super_admin > admin > viewer
	roleHierarchy := map[models.AdminRole]int{
		models.AdminRoleSuperAdmin: 3,
		models.AdminRoleAdmin:      2,
		models.AdminRoleViewer:     1,
	}

	userLevel := roleHierarchy[userRole]
	requiredLevel := roleHierarchy[requiredRole]

	return userLevel >= requiredLevel
}

// Helper functions

func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// OptionalAuth middleware що додає user в контекст якщо токен присутній
// але не вимагає його наявності
func OptionalAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Якщо токен відсутній, просто продовжуємо без auth
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			// Спробувати валідувати токен
			tokenString, err := auth.ExtractTokenFromBearer(authHeader)
			if err == nil {
				claims, err := jwtManager.ValidateToken(tokenString)
				if err == nil {
					// Додати claims в контекст
					ctx := context.WithValue(r.Context(), UserContextKey, claims)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
