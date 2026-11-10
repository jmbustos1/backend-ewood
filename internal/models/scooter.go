package models

import (
	"math"
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
	// Ultimo bloque de diagnostico del firmware ("d" de la telemetria), tal
	// cual llego. Ver esp32ewood/docs/DIAGNOSTICO.md. No se expone en la API.
	LastDiag             *string        `gorm:"type:jsonb" json:"-"`
	LastDiagAt           *time.Time     `json:"-"`
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

// IsNoFix retorna true si las coordenadas son las que envía el firmware cuando
// el GPS no tiene fix (0,0).
func IsNoFix(latitude, longitude float64) bool {
	return latitude == 0 && longitude == 0
}

// HasValidFix retorna true si el scooter tiene guardada una posición GPS real.
func (s *Scooter) HasValidFix() bool {
	return s.LastLatitude != nil && s.LastLongitude != nil &&
		!IsNoFix(*s.LastLatitude, *s.LastLongitude)
}

// Filtro de saltos imposibles: un fix que implica moverse más de
// jumpMinMeters a más de jumpMaxSpeedKmh desde el último fix recibido se
// descarta (lo aplica handler.ProcessLocationUpdate). En producción llegaron fixes mal leídos por el firmware
// (lat -36.000000 = 88 km, lon +73.000000 = otro continente).
const (
	jumpMinMeters   = 1000.0
	jumpMaxSpeedKmh = 150.0
)

// DistanceMeters aproxima la distancia entre dos puntos (equirectangular,
// suficiente para distancias de ciudad y para detectar saltos absurdos).
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0
	φ1, φ2 := lat1*math.Pi/180, lat2*math.Pi/180
	x := (lng2 - lng1) * math.Pi / 180 * math.Cos((φ1+φ2)/2)
	y := φ2 - φ1
	return math.Sqrt(x*x+y*y) * earthRadius
}

// IsImplausibleJump retorna true si pasar del fix anterior al nuevo en el
// tiempo transcurrido es imposible (más de jumpMinMeters a más de
// jumpMaxSpeedKmh). Un salto lejano pero lento se acepta (ej. scooter
// trasladado sin señal): la velocidad implícita baja con el tiempo.
func IsImplausibleJump(prevLat, prevLng float64, elapsed time.Duration, lat, lng float64) bool {
	if IsNoFix(lat, lng) || IsNoFix(prevLat, prevLng) {
		return false
	}
	dist := DistanceMeters(prevLat, prevLng, lat, lng)
	if dist < jumpMinMeters {
		return false
	}
	secs := elapsed.Seconds()
	if secs <= 0 {
		return true
	}
	return dist/secs*3.6 > jumpMaxSpeedKmh
}

// UpdateLocation actualiza la última ubicación conocida del scooter y retorna
// true si la posición cambió.
//
// Si llega 0,0 (sin fix) y el scooter ya tenía una posición real, se conserva
// la anterior: operadores necesitan la última posición conocida (T-0060).
// Un scooter "virgen" (nunca tuvo fix) sí acepta el 0,0.
// El filtro de saltos imposibles (IsImplausibleJump) lo aplica el llamador,
// que puede confirmar un salto real con fixes consecutivos.
// La batería se actualiza siempre. LastLocationUpdatedAt es la hora del
// último fix real.
func (s *Scooter) UpdateLocation(latitude, longitude float64, batteryLevel int) bool {
	s.BatteryLevel = batteryLevel

	if IsNoFix(latitude, longitude) && s.HasValidFix() {
		return false
	}
	now := time.Now()
	s.LastLatitude = &latitude
	s.LastLongitude = &longitude
	s.LastLocationUpdatedAt = &now
	return true
}

