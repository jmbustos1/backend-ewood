package models

import "time"

// PasswordReset representa un token one-shot para recuperacion de contrasena.
// Se genera al pedir /auth/forgot-password y se consume al enviar /auth/reset-password.
type PasswordReset struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	Token     string     `gorm:"uniqueIndex;not null;size:128" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}
