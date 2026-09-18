package models

import "time"

// PushToken is a browser FCM registration token for web-push notifications.
type PushToken struct {
	ID        int       `json:"id" db:"id"`
	UUID      string    `json:"uuid" db:"uuid"`
	Token     string    `json:"token" db:"token"`
	Label     string    `json:"label" db:"label"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
