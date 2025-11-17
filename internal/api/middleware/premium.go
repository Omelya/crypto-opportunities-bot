package middleware

import (
	"crypto-opportunities-bot/internal/repository"
	"encoding/json"
	"net/http"
)

// RequirePremiumMiddleware перевіряє чи користувач має активну premium підписку
func RequirePremiumMiddleware(userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Отримати claims з контексту (має бути встановлено JWT middleware)
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				respondUnauthorized(w, "Not authenticated")
				return
			}

			// Отримати користувача з БД
			user, err := userRepo.GetByID(claims.UserID)
			if err != nil || user == nil {
				respondError(w, http.StatusNotFound, "User not found")
				return
			}

			// Перевірити premium статус
			if !user.IsPremium() {
				respondPremiumRequired(w)
				return
			}

			// Якщо все ОК, продовжити
			next.ServeHTTP(w, r)
		})
	}
}

// respondPremiumRequired відповідає з помилкою що потрібен premium
func respondPremiumRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Premium subscription required",
		"message": "This feature is only available for premium users",
		"upgrade": map[string]interface{}{
			"url":     "https://t.me/yourbot?start=upgrade",
			"pricing": "Premium: $29/month",
		},
	})
}

// respondError helper для відповіді з помилкою
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
