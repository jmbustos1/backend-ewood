package models

import "testing"

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
