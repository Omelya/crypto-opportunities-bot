package handlers

import (
	"crypto-opportunities-bot/internal/version"
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

// HealthHandler обробляє health check endpoints
type HealthHandler struct{}

// NewHealthHandler створює новий HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse структура відповіді health check
type HealthResponse struct {
	Status    string            `json:"status"`
	Uptime    string            `json:"uptime"`
	Version   string            `json:"version"`
	Go        string            `json:"go_version"`
	BuildInfo map[string]string `json:"build_info,omitempty"`
}

// Health перевіряє статус системи
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime)

	response := HealthResponse{
		Status:    "healthy",
		Uptime:    uptime.String(),
		Version:   version.GetVersion(),
		Go:        runtime.Version(),
		BuildInfo: version.GetBuildInfo(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Ping простий ping endpoint
func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"pong"}`))
}
