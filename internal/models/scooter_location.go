package models

import (
	"time"

	"gorm.io/gorm"
)

// LocationEventType representa el tipo de evento de ubicación
type LocationEventType string

const (
	LocationEventTripStart      LocationEventType = "trip_start"
	LocationEventTripEnd        LocationEventType = "trip_end"
	LocationEventStatusChange   LocationEventType = "status_change"
	LocationEventPeriodicUpdate LocationEventType = "periodic_update"
)

// ScooterLocation representa una ubicación histórica del scooter (puntos clave)
type ScooterLocation struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	ScooterID   uint              `gorm:"not null;index" json:"scooter_id"`
	TripID      *uint             `gorm:"index" json:"trip_id,omitempty"`
	Latitude    float64           `gorm:"type:decimal(10,8);not null" json:"latitude"`
	Longitude   float64           `gorm:"type:decimal(11,8);not null" json:"longitude"`
	EventType   LocationEventType `gorm:"type:varchar(50);not null;index" json:"event_type"`
	BatteryLevel *int             `json:"battery_level,omitempty"`
	SpeedKmh    *float64          `gorm:"type:decimal(5,2)" json:"speed_kmh,omitempty"`
	RecordedAt  time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"recorded_at"`
	DeletedAt   gorm.DeletedAt    `gorm:"index" json:"deleted_at,omitempty"`

	// Relaciones
	Scooter Scooter `gorm:"foreignKey:ScooterID" json:"scooter,omitempty"`
	Trip    *Trip   `gorm:"foreignKey:TripID" json:"trip,omitempty"`
}

// TableName especifica el nombre de la tabla
func (ScooterLocation) TableName() string {
	return "scooter_locations"
}

