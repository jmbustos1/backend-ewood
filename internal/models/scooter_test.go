package models

import (
	"testing"
	"time"
)

func ptr(v float64) *float64 { return &v }

func TestUpdateLocation(t *testing.T) {
	tests := []struct {
		name             string
		prevLat, prevLng *float64
		lat, lng         float64
		wantLat, wantLng float64
		wantChanged      bool
	}{
		{"virgen acepta 0,0", nil, nil, 0, 0, 0, 0, true},
		{"virgen acepta fix real", nil, nil, -36.79, -73.06, -36.79, -73.06, true},
		{"en 0,0 acepta fix real", ptr(0), ptr(0), -36.79, -73.06, -36.79, -73.06, true},
		{"con fix ignora 0,0", ptr(-36.79), ptr(-73.06), 0, 0, -36.79, -73.06, false},
		{"con fix acepta fix nuevo", ptr(-36.79), ptr(-73.06), -36.80, -73.07, -36.80, -73.07, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Scooter{LastLatitude: tt.prevLat, LastLongitude: tt.prevLng, BatteryLevel: 10}
			changed := s.UpdateLocation(tt.lat, tt.lng, 80)

			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			if s.BatteryLevel != 80 {
				t.Errorf("battery = %d, want 80 (se actualiza siempre)", s.BatteryLevel)
			}
			if *s.LastLatitude != tt.wantLat || *s.LastLongitude != tt.wantLng {
				t.Errorf("pos = (%v, %v), want (%v, %v)", *s.LastLatitude, *s.LastLongitude, tt.wantLat, tt.wantLng)
			}
		})
	}
}

func TestImplausibleJump(t *testing.T) {
	const lat, lng = -36.797092, -73.059965
	tests := []struct {
		name     string
		elapsed  time.Duration
		lat, lng float64
		want     bool
	}{
		{"drift de metros", 6 * time.Second, -36.797120, -73.060010, false},
		{"lat sin minutos (prod 09/11)", 6 * time.Second, -36.000000, -73.059989, true},
		{"lon sin signo (prod 09/11)", 6 * time.Second, -36.797110, 73.000000, true},
		{"200 m en 6 s (bajo el umbral de distancia)", 6 * time.Second, -36.795300, -73.059965, false},
		{"5 km en 6 s", 6 * time.Second, -36.752000, -73.059965, true},
		{"5 km en 10 min (trasladado)", 10 * time.Minute, -36.752000, -73.059965, false},
		{"0,0 no es salto", 6 * time.Second, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImplausibleJump(lat, lng, tt.elapsed, tt.lat, tt.lng); got != tt.want {
				t.Errorf("IsImplausibleJump = %v, want %v", got, tt.want)
			}
		})
	}
}
