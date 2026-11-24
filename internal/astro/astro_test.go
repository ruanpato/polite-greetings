package astro

import (
	"math"
	"testing"
	"time"
)

func utc(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func near(got, want, tolerance float64) bool { return math.Abs(got-want) <= tolerance }

func TestSubsolar(t *testing.T) {
	tests := []struct {
		at                 string
		lat, lon, tolerate float64
	}{
		{"2026-06-21T12:00:00Z", 23.44, 0.4, 1},   // June solstice; equation of time near -2 min
		{"2026-12-21T12:00:00Z", -23.44, -0.4, 1}, // December solstice; equation of time near +2 min
		{"2026-11-24T12:00:00Z", -20.6, -3.3, 1},  // the sun is about 13 min fast in late November
		{"2026-11-24T18:00:00Z", -20.7, -93.3, 1}, // six hours later, 90° further west
	}
	for _, tt := range tests {
		lat, lon := Subsolar(utc(tt.at))
		if !near(lat, tt.lat, tt.tolerate) || !near(lon, tt.lon, tt.tolerate) {
			t.Errorf("Subsolar(%s) = %.2f, %.2f; want %.2f, %.2f", tt.at, lat, lon, tt.lat, tt.lon)
		}
	}
	// March equinox, 20 March 2026 at about 14:46 UTC.
	if lat, _ := Subsolar(utc("2026-03-20T14:46:00Z")); !near(lat, 0, 0.2) {
		t.Errorf("declination at the equinox = %.2f, want about 0", lat)
	}
}

func TestMoon(t *testing.T) {
	tests := []struct {
		name   string
		at     string
		phase  Phase
		waxing bool
	}{
		{"total solar eclipse, a new moon", "2026-08-12T17:46:00Z", NewMoon, false},
		{"four days later", "2026-08-16T18:00:00Z", WaxingCrescent, true},
		{"total lunar eclipse, a full moon", "2026-03-03T11:33:00Z", FullMoon, false},
		{"full moon of 24 November", "2026-11-24T12:00:00Z", FullMoon, false},
		{"a week after it", "2026-12-01T12:00:00Z", LastQuarter, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MoonAt(utc(tt.at))
			if m.Phase() != tt.phase {
				t.Errorf("phase = %s (%.3f lit, waxing %v), want %s", m.Phase(), m.Illumination, m.Waxing, tt.phase)
			}
			if tt.phase != NewMoon && tt.phase != FullMoon && m.Waxing != tt.waxing {
				t.Errorf("waxing = %v, want %v", m.Waxing, tt.waxing)
			}
		})
	}
}

// TestMoonCycle walks two months in six-hour steps: the lit share and the age
// must agree, and the age must only grow, wrapping at new moon.
func TestMoonCycle(t *testing.T) {
	prev := MoonAt(utc("2026-01-01T00:00:00Z"))
	wraps := 0
	for at := utc("2026-01-01T06:00:00Z"); at.Before(utc("2026-03-01T00:00:00Z")); at = at.Add(6 * time.Hour) {
		m := MoonAt(at)
		if m.Illumination < 0 || m.Illumination > 1 || m.Age < 0 || m.Age > 1 {
			t.Fatalf("%s: out of range: %+v", at, m)
		}
		if lit := (1 - math.Cos(2*math.Pi*m.Age)) / 2; !near(lit, m.Illumination, 1e-9) {
			t.Fatalf("%s: age %.3f implies %.3f lit, got %.3f", at, m.Age, lit, m.Illumination)
		}
		if m.Age < prev.Age {
			wraps++
			if prev.Age < 0.95 || m.Age > 0.05 {
				t.Fatalf("%s: age went back from %.3f to %.3f", at, prev.Age, m.Age)
			}
		}
		prev = m
	}
	if wraps != 2 {
		t.Errorf("new moons in January and February 2026 = %d, want 2", wraps)
	}
}
