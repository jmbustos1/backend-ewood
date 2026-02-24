package models

import (
	"time"

	"gorm.io/gorm"
)

// ScooterStatusHistory representa el historial de cambios de estado de un scooter
type ScooterStatusHistory struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ScooterID  uint           `gorm:"not null;index" json:"scooter_id"`
	OldStatus  *string        `gorm:"type:varchar(20)" json:"old_status,omitempty"`
	NewStatus  string         `gorm:"type:varchar(20);not null" json:"new_status"`
	Reason     *string        `gorm:"type:text" json:"reason,omitempty"`
	ChangedAt  time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"changed_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relaciones
	Scooter Scooter `gorm:"foreignKey:ScooterID" json:"scooter,omitempty"`
}

// TableName especifica el nombre de la tabla
func (ScooterStatusHistory) TableName() string {
	return "scooter_status_history"
}

