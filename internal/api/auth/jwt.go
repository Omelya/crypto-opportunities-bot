package auth

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims представляє JWT claims
type Claims struct {
	UserID   uint              `json:"user_id"`
	Username string            `json:"username"`
	Role     models.AdminRole  `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager керує JWT токенами
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager створює новий JWTManager
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}
}

// GenerateToken генерує JWT токен для адміністратора
func (m *JWTManager) GenerateToken(admin *models.AdminUser) (string, error) {
	claims := Claims{
		UserID:   admin.ID,
		Username: admin.Username,
		Role:     admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "crypto-opportunities-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

// ValidateToken валідує JWT токен і повертає claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Перевіряємо метод підпису
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.secretKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RefreshToken оновлює токен (якщо він валідний)
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Генеруємо новий токен з тими ж даними
	newClaims := Claims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "crypto-opportunities-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return token.SignedString([]byte(m.secretKey))
}

// ExtractTokenFromBearer витягує токен з Authorization header
func ExtractTokenFromBearer(authHeader string) (string, error) {
	const bearerPrefix = "Bearer "

	if len(authHeader) < len(bearerPrefix) {
		return "", fmt.Errorf("invalid authorization header")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	return authHeader[len(bearerPrefix):], nil
}
