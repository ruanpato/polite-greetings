// Package astro tells where the sun is overhead and how the moon looks, at a
// moment. It uses the low-precision formulas of the Astronomical Almanac, good
// to about a degree: plenty for a card, not for navigation.
package astro

import (
	"math"
	"time"
)

const deg = math.Pi / 180

var j2000 = time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)

// days is the number of days since 2000-01-01 12:00 UTC.
func days(t time.Time) float64 {
	return float64(t.Sub(j2000)) / float64(24*time.Hour)
}

// sun returns the sun's mean longitude, in degrees, and its apparent ecliptic
// longitude, in radians, n days after J2000.
func sun(n float64) (meanLongitude, eclipticLongitude float64) {
	l := 280.460 + 0.9856474*n
	g := (357.528 + 0.9856003*n) * deg
	return l, (l + 1.915*math.Sin(g) + 0.020*math.Sin(2*g)) * deg
}

// Subsolar returns the latitude and longitude, in degrees, of the point where
// the sun is overhead at t.
func Subsolar(t time.Time) (lat, lon float64) {
	n := days(t)
	l, lambda := sun(n)
	eps := (23.439 - 0.0000004*n) * deg
	dec := math.Asin(math.Sin(eps) * math.Sin(lambda))
	ra := math.Atan2(math.Cos(eps)*math.Sin(lambda), math.Cos(lambda))
	equationOfTime := Wrap180(l - ra/deg) // degrees of the Earth's turn
	u := t.UTC()
	hours := float64(u.Hour()) + float64(u.Minute())/60 + float64(u.Second())/3600 + float64(u.Nanosecond())/3.6e12
	return dec / deg, Wrap180(-15*(hours-12) - equationOfTime)
}

// Phase names the moon by how much of it is lit.
type Phase string

// The eight phases.
const (
	NewMoon        Phase = "new-moon"
	WaxingCrescent Phase = "waxing-crescent"
	FirstQuarter   Phase = "first-quarter"
	WaxingGibbous  Phase = "waxing-gibbous"
	FullMoon       Phase = "full-moon"
	WaningGibbous  Phase = "waning-gibbous"
	LastQuarter    Phase = "last-quarter"
	WaningCrescent Phase = "waning-crescent"
)

// Phases lists the phases in the order of a cycle.
var Phases = []Phase{NewMoon, WaxingCrescent, FirstQuarter, WaxingGibbous, FullMoon, WaningGibbous, LastQuarter, WaningCrescent}

// Moon is the moon as seen from the Earth.
type Moon struct {
	Illumination float64 // share of the disk that is lit, from 0 to 1
	Waxing       bool    // the lit share grows: from new moon to full moon
	Age          float64 // share of the cycle since new moon, from 0 to 1; 0.5 is full
}

// MoonAt returns the moon at t.
func MoonAt(t time.Time) Moon {
	n := days(t)
	_, sunLongitude := sun(n)
	// The moon's mean longitude, mean anomaly and argument of latitude, with the
	// largest term of each correction.
	meanLongitude := (218.316 + 13.176396*n) * deg
	anomaly := (134.963 + 13.064993*n) * deg
	argument := (93.272 + 13.229350*n) * deg
	longitude := meanLongitude + 6.289*deg*math.Sin(anomaly)
	latitude := 5.128 * deg * math.Sin(argument)
	distance := 385001 - 20905*math.Cos(anomaly) // km
	const sunDistance = 149598000.0              // km

	elongation := math.Acos(math.Cos(latitude) * math.Cos(longitude-sunLongitude))
	phaseAngle := math.Atan2(sunDistance*math.Sin(elongation), distance-sunDistance*math.Cos(elongation))
	m := Moon{Illumination: (1 + math.Cos(phaseAngle)) / 2, Waxing: math.Sin(longitude-sunLongitude) > 0}
	if m.Waxing {
		m.Age = 0.5 - phaseAngle/(2*math.Pi)
	} else {
		m.Age = 0.5 + phaseAngle/(2*math.Pi)
	}
	return m
}

// Phase names the moon by its lit share: the new and full moons and the
// quarters are moments, so they get a small window around their share.
func (m Moon) Phase() Phase {
	k := m.Illumination
	switch {
	case k < 0.03:
		return NewMoon
	case k > 0.97:
		return FullMoon
	case math.Abs(k-0.5) < 0.05:
		return pick(m.Waxing, FirstQuarter, LastQuarter)
	case k < 0.5:
		return pick(m.Waxing, WaxingCrescent, WaningCrescent)
	default:
		return pick(m.Waxing, WaxingGibbous, WaningGibbous)
	}
}

func pick(waxing bool, a, b Phase) Phase {
	if waxing {
		return a
	}
	return b
}

// Wrap180 brings an angle in degrees into [-180, 180).
func Wrap180(x float64) float64 {
	x = math.Mod(x+180, 360)
	if x < 0 {
		x += 360
	}
	return x - 180
}
