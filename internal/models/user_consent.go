package models

import "time"

// ConsentDocument enum de tipos de documento consentidos
const (
	ConsentTerms   = "terms"
	ConsentPrivacy = "privacy"
)

// UserConsent es un registro append-only del consentimiento del usuario a un
// documento (Terminos o Privacidad) en una version dada. Cada re-aceptacion
// (por subida de version del documento) crea una nueva fila.
type UserConsent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	Document   string    `gorm:"not null;size:32" json:"document"` // ConsentTerms | ConsentPrivacy
	Version    string    `gorm:"not null;size:32" json:"version"`
	AcceptedAt time.Time `gorm:"not null" json:"accepted_at"`
	IPAddress  string    `gorm:"size:64" json:"ip_address,omitempty"`
	UserAgent  string    `gorm:"size:255" json:"user_agent,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}
