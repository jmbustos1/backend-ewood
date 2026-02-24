package models

import (
	"time"

	"gorm.io/gorm"
)

// User representa un usuario del sistema
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Name      string         `gorm:"not null;size:255" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relaciones
	Scooters []Scooter `gorm:"foreignKey:CurrentUserID" json:"scooters,omitempty"`
	Trips    []Trip    `gorm:"foreignKey:UserID" json:"trips,omitempty"`
}

// TableName especifica el nombre de la tabla
func (User) TableName() string {
	return "users"
}

