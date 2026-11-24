// Package greeting decides which greeting fits a place at a moment, from the
// sun and the local clock. It knows nothing about HTTP or SVG.
//
// The rules:
//   - Morning runs from sunrise until 12:00 on the local clock.
//   - Afternoon runs from 12:00 until sunset.
//   - Evening runs from sunset until the next sunrise.
//   - On a day when the sun does not rise or set (polar night, midnight sun),
//     the clock decides: 06:00, 12:00 and 18:00.
package greeting

import (
	"slices"
	"time"

	"github.com/nathan-osman/go-sunrise"
)

// Period is the part of the day a greeting belongs to.
type Period string

// The three periods of a day.
const (
	Morning   Period = "morning"
	Afternoon Period = "afternoon"
	Evening   Period = "evening"
)

// SunState tells whether the sun rises and sets on a local day.
type SunState string

// The three states of the sun on a day.
const (
	SunNormal   SunState = "normal"
	PolarNight  SunState = "polar-night"
	MidnightSun SunState = "midnight-sun"
)

// horizon is the sun's elevation at sunrise and sunset, in degrees, with
// refraction, as go-sunrise uses it.
const horizon = -0.833

// Day is the sun on one local calendar day.
type Day struct {
	State   SunState
	Sunrise time.Time // zero unless State is SunNormal
	Sunset  time.Time // zero unless State is SunNormal
}

// Daylight is the time the sun is up on the day.
func (d Day) Daylight() time.Duration {
	switch d.State {
	case SunNormal:
		return d.Sunset.Sub(d.Sunrise)
	case MidnightSun:
		return 24 * time.Hour
	default:
		return 0
	}
}

// Result is the greeting for a place at a moment.
type Result struct {
	Now        time.Time // the moment, in the place's time zone
	Period     Period
	NextChange time.Time // when Period changes next, in the place's time zone
	Today      Day       // the sun on the local calendar day of Now
}

// Progress returns the share of today's daylight that has passed. It returns
// false when the sun is down or does not rise and set today.
func (r Result) Progress() (float64, bool) {
	t := r.Today
	if t.State != SunNormal || r.Now.Before(t.Sunrise) || !r.Now.Before(t.Sunset) {
		return 0, false
	}
	return float64(r.Now.Sub(t.Sunrise)) / float64(t.Daylight()), true
}

// Compute returns the greeting at the moment now for the place at latitude lat
// and longitude lon, whose clock follows loc.
func Compute(now time.Time, loc *time.Location, lat, lon float64) Result {
	now = now.In(loc)
	y, m, d := now.Date()

	// Boundaries of yesterday, today and the next two days: yesterday's sunset
	// can fall after midnight, and tomorrow always holds the next change.
	var bounds []boundary
	for offset := -1; offset <= 2; offset++ {
		dy, dm, dd := time.Date(y, m, d+offset, 12, 0, 0, 0, loc).Date()
		bounds = append(bounds, boundaries(SunOn(lat, lon, loc, dy, dm, dd), loc, dy, dm, dd)...)
	}
	slices.SortStableFunc(bounds, func(a, b boundary) int { return a.at.Compare(b.at) })

	res := Result{Now: now, Period: Evening, Today: SunOn(lat, lon, loc, y, m, d)}
	for _, b := range bounds {
		if b.at.After(now) {
			res.NextChange = b.at
			break
		}
		res.Period = b.period
	}
	return res
}

// SunOn returns the sun on the local calendar day y-m-d at latitude lat and
// longitude lon, with times in loc.
func SunOn(lat, lon float64, loc *time.Location, y int, m time.Month, d int) Day {
	// go-sunrise works on solar days: the date it takes is the one whose solar
	// noon, in UTC, it computes. Where the clock is far from solar time (UTC+14
	// in Kiritimati, at 157°W) that date is not the local one, so pick the solar
	// day whose noon falls nearest to 12:00 on the local day.
	localNoon := time.Date(y, m, d, 12, 0, 0, 0, loc).UTC()
	sy, sm, sd := localNoon.Add(time.Duration(lon / 15 * float64(time.Hour))).Date()

	rise, set := sunrise.SunriseSunset(lat, lon, sy, sm, sd)
	if !rise.IsZero() && !set.IsZero() {
		return Day{State: SunNormal, Sunrise: rise.In(loc), Sunset: set.In(loc)}
	}
	// No sunrise or sunset: the sun is either up or down all day. Its
	// elevation at solar noon tells which.
	noon := sunrise.JulianDayToTime(sunrise.MeanSolarNoon(lon, sy, sm, sd))
	if sunrise.Elevation(lat, lon, noon) > horizon {
		return Day{State: MidnightSun}
	}
	return Day{State: PolarNight}
}

// boundary is a moment when the period changes.
type boundary struct {
	at     time.Time
	period Period // the period that starts at this moment
}

// boundaries lists the period changes of one local day, in time order.
func boundaries(day Day, loc *time.Location, y int, m time.Month, d int) []boundary {
	clock := func(hour int) time.Time { return time.Date(y, m, d, hour, 0, 0, 0, loc) }
	if day.State != SunNormal {
		return []boundary{{clock(6), Morning}, {clock(12), Afternoon}, {clock(18), Evening}}
	}
	noon := clock(12)
	switch {
	case !day.Sunrise.Before(noon): // the sun rises after noon: no morning
		return []boundary{{day.Sunrise, Afternoon}, {day.Sunset, Evening}}
	case !day.Sunset.After(noon): // the sun sets before noon: no afternoon
		return []boundary{{day.Sunrise, Morning}, {day.Sunset, Evening}}
	default:
		return []boundary{{day.Sunrise, Morning}, {noon, Afternoon}, {day.Sunset, Evening}}
	}
}
