package models

import (
	"time"

	"gorm.io/gorm"
)

// TripStatus representa los estados posibles de un viaje
type TripStatus string

const (
	TripStatusActive    TripStatus = "active"
	TripStatusCompleted TripStatus = "completed"
	TripStatusCancelled TripStatus = "cancelled"
)

// Trip representa un viaje en scooter
type Trip struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ScooterID      uint           `gorm:"not null;index" json:"scooter_id"`
	UserID         uint           `gorm:"not null;index" json:"user_id"`
	StartLatitude  float64        `gorm:"type:decimal(10,8);not null" json:"start_latitude"`
	StartLongitude float64        `gorm:"type:decimal(11,8);not null" json:"start_longitude"`
	EndLatitude    *float64       `gorm:"type:decimal(10,8)" json:"end_latitude,omitempty"`
	EndLongitude   *float64       `gorm:"type:decimal(11,8)" json:"end_longitude,omitempty"`
	StartedAt      time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"started_at"`
	EndedAt        *time.Time     `json:"ended_at,omitempty"`
	DurationSeconds *int          `json:"duration_seconds,omitempty"`
	DistanceMeters *float64       `gorm:"type:decimal(10,2)" json:"distance_meters,omitempty"`
	AverageSpeedKmh *float64      `gorm:"type:decimal(5,2)" json:"average_speed_kmh,omitempty"`
	MaxSpeedKmh     *float64      `gorm:"type:decimal(5,2)" json:"max_speed_kmh,omitempty"`
	Status         TripStatus     `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	CostAmount     *float64       `gorm:"type:decimal(10,2)" json:"cost_amount,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relaciones
	Scooter  Scooter          `gorm:"foreignKey:ScooterID" json:"scooter,omitempty"`
	User     User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Locations []ScooterLocation `gorm:"foreignKey:TripID" json:"locations,omitempty"`
}

// TableName especifica el nombre de la tabla
func (Trip) TableName() string {
	return "trips"
}

// IsActive retorna true si el viaje está activo
func (t *Trip) IsActive() bool {
	return t.Status == TripStatusActive
}

// CalculateDuration calcula la duración del viaje en segundos
func (t *Trip) CalculateDuration() {
	if t.EndedAt != nil {
		duration := int(t.EndedAt.Sub(t.StartedAt).Seconds())
		t.DurationSeconds = &duration
	}
}

// Complete finaliza el viaje con las coordenadas de fin
func (t *Trip) Complete(endLat, endLng float64, distance, avgSpeed, maxSpeed, cost float64) {
	now := time.Now()
	t.EndLatitude = &endLat
	t.EndLongitude = &endLng
	t.EndedAt = &now
	t.Status = TripStatusCompleted
	t.DistanceMeters = &distance
	t.AverageSpeedKmh = &avgSpeed
	t.MaxSpeedKmh = &maxSpeed
	t.CostAmount = &cost
	t.CalculateDuration()
}

