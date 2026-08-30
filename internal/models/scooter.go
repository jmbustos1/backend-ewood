package models

import (
	"time"

	"gorm.io/gorm"
)

// ScooterStatus representa los estados posibles de un scooter
type ScooterStatus string

const (
	ScooterStatusAvailable  ScooterStatus = "available"
	ScooterStatusInUse      ScooterStatus = "in_use"
	ScooterStatusMaintenance ScooterStatus = "maintenance"
	ScooterStatusOffline    ScooterStatus = "offline"
)

// LockState representa el estado fisico del actuador reportado por el firmware
// (byte Locker del frame SYS STATUS del hub de sensores).
type LockState string

const (
	LockStateUnknown  LockState = "unknown"
	LockStateLocked   LockState = "locked"
	LockStateUnlocked LockState = "unlocked"
)

// Scooter representa el estado actual de un scooter
type Scooter struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	Code                 string         `gorm:"uniqueIndex;not null;size:50" json:"code"`
	Model                string         `gorm:"size:100" json:"model"`
	Status               ScooterStatus  `gorm:"type:varchar(20);not null;default:'available'" json:"status"`
	BatteryLevel         int            `gorm:"not null;default:100;check:battery_level >= 0 AND battery_level <= 100" json:"battery_level"`
	LastLatitude         *float64       `gorm:"type:decimal(10,8)" json:"last_latitude,omitempty"`
	LastLongitude        *float64       `gorm:"type:decimal(11,8)" json:"last_longitude,omitempty"`
	LastLocationUpdatedAt *time.Time    `json:"last_location_updated_at,omitempty"`
	UnlockCode           *string        `gorm:"uniqueIndex;size:50" json:"-"`
	CityID               *uint          `json:"city_id,omitempty"`
	CurrentUserID        *uint          `json:"current_user_id,omitempty"`
	CurrentTripID        *uint          `json:"current_trip_id,omitempty"`
	TripStartedAt        *time.Time     `json:"trip_started_at,omitempty"`
	LockState            LockState      `gorm:"type:varchar(16);default:'unknown'" json:"lock_state"`
	LockStateUpdatedAt   *time.Time     `json:"lock_state_updated_at,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relaciones
	City        *City  `gorm:"foreignKey:CityID" json:"city,omitempty"`
	CurrentUser *User  `gorm:"foreignKey:CurrentUserID" json:"current_user,omitempty"`
	CurrentTrip *Trip  `gorm:"foreignKey:CurrentTripID" json:"current_trip,omitempty"`
	Trips       []Trip `gorm:"foreignKey:ScooterID" json:"trips,omitempty"`
	Locations   []ScooterLocation `gorm:"foreignKey:ScooterID" json:"locations,omitempty"`
	StatusHistory []ScooterStatusHistory `gorm:"foreignKey:ScooterID" json:"status_history,omitempty"`
}

// TableName especifica el nombre de la tabla
func (Scooter) TableName() string {
	return "scooters"
}

// IsAvailable retorna true si el scooter está disponible
func (s *Scooter) IsAvailable() bool {
	return s.Status == ScooterStatusAvailable
}

// IsInUse retorna true si el scooter está en uso
func (s *Scooter) IsInUse() bool {
	return s.Status == ScooterStatusInUse
}

// UpdateLocation actualiza la última ubicación conocida del scooter
func (s *Scooter) UpdateLocation(latitude, longitude float64, batteryLevel int) {
	now := time.Now()
	s.LastLatitude = &latitude
	s.LastLongitude = &longitude
	s.LastLocationUpdatedAt = &now
	s.BatteryLevel = batteryLevel
}

