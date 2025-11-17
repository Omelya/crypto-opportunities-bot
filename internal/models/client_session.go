package models

import (
	"time"
)

// ClientSession представляє активну сесію Premium клієнта
type ClientSession struct {
	BaseModel

	UserID uint  `gorm:"index;not null" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	SessionID    string `gorm:"uniqueIndex;not null" json:"session_id"` // UUID
	ConnectionID string `gorm:"index" json:"connection_id"`             // WebSocket connection ID

	// Client info
	ClientVersion string `json:"client_version"` // "1.0.0"
	Platform      string `json:"platform"`       // "windows", "linux", "macos"
	IPAddress     string `json:"ip_address"`

	// Status
	IsActive       bool       `gorm:"default:true;index" json:"is_active"`
	LastHeartbeat  time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"last_heartbeat"`
	ConnectedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
}

func (*ClientSession) TableName() string {
	return "client_sessions"
}

// IsAlive перевіряє чи сесія ще активна (heartbeat < 2 хв тому)
func (cs *ClientSession) IsAlive() bool {
	if !cs.IsActive {
		return false
	}

	// Якщо не отримували heartbeat більше 2 хвилин - вважаємо мертвою
	return time.Since(cs.LastHeartbeat) < 2*time.Minute
}

// Duration скільки часу клієнт підключений
func (cs *ClientSession) Duration() time.Duration {
	if cs.DisconnectedAt != nil {
		return cs.DisconnectedAt.Sub(cs.ConnectedAt)
	}
	return time.Since(cs.ConnectedAt)
}
