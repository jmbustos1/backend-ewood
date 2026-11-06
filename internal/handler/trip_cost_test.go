package handler

import "testing"

func TestCalculateTripCost(t *testing.T) {
	t.Setenv("PRICE_PER_MINUTE", "120")

	cases := []struct {
		name     string
		seconds  int
		minutes  int
		subtotal float64
		tax      float64
		total    float64
	}{
		{"cero segundos no cobra", 0, 0, 0, 0, 0},
		{"1 segundo cobra 1 minuto", 1, 1, 120, 23, 143},
		{"60 segundos exactos cobra 1 minuto", 60, 1, 120, 23, 143},
		{"65 segundos cobra 2 minutos", 65, 2, 240, 46, 286},
		{"10 minutos", 600, 10, 1200, 228, 1428},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := calculateTripCost(c.seconds)
			if got.MinutesCharged != c.minutes || got.Subtotal != c.subtotal || got.TaxAmount != c.tax || got.Total != c.total {
				t.Errorf("calculateTripCost(%d) = %+v; want minutes=%d subtotal=%v tax=%v total=%v",
					c.seconds, got, c.minutes, c.subtotal, c.tax, c.total)
			}
		})
	}
}
