package handler

import (
	"testing"
	"time"
)

func TestCheckFix(t *testing.T) {
	const id = 9001
	t.Cleanup(func() { delete(fixTracks, id) })
	t0 := time.Date(2026, 11, 10, 1, 0, 0, 0, time.UTC)
	at := func(s int) time.Time { return t0.Add(time.Duration(s) * time.Second) }

	steps := []struct {
		name               string
		sec                int
		lat, lng           float64
		accepted, confirms bool
	}{
		{"primer fix se acepta", 0, -36.797092, -73.059965, true, false},
		{"drift de metros", 6, -36.797120, -73.060010, true, false},
		{"lat sin minutos (prod 09/11)", 12, -36.000000, -73.059989, false, false},
		{"vuelve a la base", 18, -36.797110, -73.060000, true, false},
		{"lon sin signo (prod 09/11)", 24, -36.797110, 73.000000, false, false},
		{"0,0 no se filtra", 30, 0, 0, true, false},
		{"traslado real: 1er fix lejano", 36, -36.700000, -73.100000, false, false},
		{"traslado real: 2do fix confirma", 42, -36.700020, -73.100010, true, true},
		{"5 km tras 10 min se acepta", 642, -36.745000, -73.100000, true, false},
	}
	for _, st := range steps {
		acc, conf := checkFix(id, st.lat, st.lng, at(st.sec))
		if acc != st.accepted || conf != st.confirms {
			t.Errorf("%s: accepted=%v confirmed=%v, want %v %v", st.name, acc, conf, st.accepted, st.confirms)
		}
	}
}
