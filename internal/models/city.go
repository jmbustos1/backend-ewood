package models

import (
	"time"

	"gorm.io/gorm"
)

type City struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null;size:100" json:"name"`
	Country   string         `gorm:"not null;size:100;default:Chile" json:"country"`
	Lat       float64        `gorm:"type:decimal(10,8);not null" json:"lat"`
	Lng       float64        `gorm:"type:decimal(11,8);not null" json:"lng"`
	RadiusKm  float64        `gorm:"type:decimal(5,2);not null;default:5.0" json:"radius_km"`
	Active    bool           `gorm:"not null;default:true" json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (City) TableName() string {
	return "cities"
}
