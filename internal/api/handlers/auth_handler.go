package handlers

import (
	"crypto-opportunities-bot/internal/api/auth"
	"crypto-opportunities-bot/internal/api/middleware"
	"crypto-opportunities-bot/internal/repository"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// AuthHandler обробляє authentication запити
type AuthHandler struct {
	adminRepo  repository.AdminRepository
	jwtManager *auth.JWTManager
}

// NewAuthHandler створює новий AuthHandler
func NewAuthHandler(adminRepo repository.AdminRepository, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		adminRepo:  adminRepo,
		jwtManager: jwtManager,
	}
}

// LoginRequest структура запиту для login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse структура відповіді login
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"` // seconds
	User      AdminUserResponse `json:"user"`
}

// AdminUserResponse структура для відповіді з даними користувача
type AdminUserResponse struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	IsActive    bool   `json:"is_active"`
	LastLoginAt string `json:"last_login_at,omitempty"`
}

// Login аутентифікує адміністратора
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		respondError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Знайти адміністратора
	admin, err := h.adminRepo.GetByUsername(req.Username)
	if err != nil {
		log.Printf("⚠️ Login attempt for non-existent user: %s", req.Username)
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Перевірити чи активний
	if !admin.IsActive {
		log.Printf("⚠️ Login attempt for inactive user: %s", req.Username)
		respondError(w, http.StatusUnauthorized, "Account is disabled")
		return
	}

	// Перевірити пароль
	if !admin.CheckPassword(req.Password) {
		log.Printf("⚠️ Failed login attempt for user: %s", req.Username)
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Генерувати JWT токен
	token, err := h.jwtManager.GenerateToken(admin)
	if err != nil {
		log.Printf("❌ Failed to generate token: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Оновити last login
	if err := h.adminRepo.UpdateLastLogin(admin.ID); err != nil {
		log.Printf("⚠️ Failed to update last login: %v", err)
	}

	log.Printf("✅ User logged in: %s (role: %s)", admin.Username, admin.Role)

	// Відповісти з токеном
	response := LoginResponse{
		Token:     token,
		ExpiresIn: 24 * 60 * 60, // 24 години в секундах
		User: AdminUserResponse{
			ID:       admin.ID,
			Username: admin.Username,
			Email:    admin.Email,
			Role:     string(admin.Role),
			IsActive: admin.IsActive,
		},
	}

	if admin.LastLoginAt != nil {
		response.User.LastLoginAt = admin.LastLoginAt.String()
	}

	respondJSON(w, http.StatusOK, response)
}

// Logout завершує сесію (на клієнті потрібно видалити токен)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// JWT токени stateless, тому logout виконується на клієнті
	// Але можемо додати в blacklist якщо потрібно
	// TODO: Implement token blacklist if needed

	claims := middleware.GetUserFromContext(r.Context())
	if claims != nil {
		log.Printf("🚪 User logged out: %s", claims.Username)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// Me повертає інформацію про поточного користувача
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Отримати повні дані користувача з БД
	admin, err := h.adminRepo.GetByID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Перевірити чи активний
	if !admin.IsActive {
		respondError(w, http.StatusUnauthorized, "Account is disabled")
		return
	}

	response := AdminUserResponse{
		ID:       admin.ID,
		Username: admin.Username,
		Email:    admin.Email,
		Role:     string(admin.Role),
		IsActive: admin.IsActive,
	}

	if admin.LastLoginAt != nil {
		response.LastLoginAt = admin.LastLoginAt.String()
	}

	respondJSON(w, http.StatusOK, response)
}

// RefreshToken оновлює JWT токен
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Отримати поточний токен
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondError(w, http.StatusUnauthorized, "Missing authorization header")
		return
	}

	tokenString, err := auth.ExtractTokenFromBearer(authHeader)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid authorization header")
		return
	}

	// Оновити токен
	newToken, err := h.jwtManager.RefreshToken(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token":      newToken,
		"expires_in": 24 * 60 * 60,
	})
}
