package greeting

import (
	"math/rand/v2"
	"testing"
	"time"
)

type place struct {
	name     string
	lat, lon float64
	tz       string
}

var (
	brasilia  = place{"Brasília", -15.78, -47.93, "America/Sao_Paulo"}
	tokyo     = place{"Tokyo", 35.68, 139.69, "Asia/Tokyo"}
	sydney    = place{"Sydney", -33.87, 151.21, "Australia/Sydney"}
	lisbon    = place{"Lisbon", 38.72, -9.14, "Europe/Lisbon"}
	tromso    = place{"Tromsø", 69.65, 18.96, "Europe/Oslo"}
	quito     = place{"Quito", -0.18, -78.47, "America/Guayaquil"}
	kiritimat = place{"Kiritimati", 1.87, -157.43, "Pacific/Kiritimati"}
	pagoPago  = place{"Pago Pago", -14.28, -170.70, "Pacific/Pago_Pago"}
	kolkata   = place{"Kolkata", 22.57, 88.36, "Asia/Kolkata"}
	kathmandu = place{"Kathmandu", 27.72, 85.32, "Asia/Kathmandu"}
)

func (p place) loc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(p.tz)
	if err != nil {
		t.Fatalf("load %s: %v", p.tz, err)
	}
	return loc
}

// at builds a local wall-clock time at the place.
func (p place) at(t *testing.T, y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, p.loc(t))
}

func TestComputePeriod(t *testing.T) {
	tests := []struct {
		name  string
		place place
		when  [5]int // month, day, hour, minute in 2026; index 0 unused
		want  Period
		state SunState
	}{
		// v1 answered "Good Afternoon" to these two mornings.
		{"Tokyo at 08:00", tokyo, [5]int{0, 11, 24, 8, 0}, Morning, SunNormal},
		{"Sydney at 09:00", sydney, [5]int{0, 11, 24, 9, 0}, Morning, SunNormal},
		{"Brasília before dawn", brasilia, [5]int{0, 11, 24, 1, 0}, Evening, SunNormal},
		{"Brasília at 09:00", brasilia, [5]int{0, 11, 24, 9, 0}, Morning, SunNormal},
		{"Brasília at 15:00", brasilia, [5]int{0, 11, 24, 15, 0}, Afternoon, SunNormal},
		{"Brasília at 21:04", brasilia, [5]int{0, 11, 24, 21, 4}, Evening, SunNormal},
		{"Lisbon on the spring change", lisbon, [5]int{0, 3, 29, 11, 30}, Morning, SunNormal},
		{"Lisbon on the autumn change", lisbon, [5]int{0, 10, 25, 12, 30}, Afternoon, SunNormal},
		// v1 crashed with a 500 here.
		{"Tromsø, polar night, 09:00", tromso, [5]int{0, 12, 15, 9, 0}, Morning, PolarNight},
		{"Tromsø, polar night, 12:00", tromso, [5]int{0, 12, 15, 12, 0}, Afternoon, PolarNight},
		{"Tromsø, midnight sun, 01:00", tromso, [5]int{0, 6, 21, 1, 0}, Evening, MidnightSun},
		{"Quito at 13:00", quito, [5]int{0, 11, 24, 13, 0}, Afternoon, SunNormal},
		{"Kiritimati (UTC+14) at 08:00", kiritimat, [5]int{0, 11, 24, 8, 0}, Morning, SunNormal},
		{"Pago Pago (UTC-11) at 20:00", pagoPago, [5]int{0, 11, 24, 20, 0}, Evening, SunNormal},
		{"Kolkata (UTC+5:30) at 17:30", kolkata, [5]int{0, 11, 24, 17, 30}, Evening, SunNormal},
		{"Kathmandu (UTC+5:45) at 07:00", kathmandu, [5]int{0, 11, 24, 7, 0}, Morning, SunNormal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := tt.place.at(t, 2026, time.Month(tt.when[1]), tt.when[2], tt.when[3], tt.when[4])
			got := Compute(now, tt.place.loc(t), tt.place.lat, tt.place.lon)
			if got.Period != tt.want {
				t.Errorf("period = %s, want %s (sunrise %s, sunset %s)", got.Period, tt.want,
					got.Today.Sunrise.Format("15:04"), got.Today.Sunset.Format("15:04"))
			}
			if got.Today.State != tt.state {
				t.Errorf("sun state = %s, want %s", got.Today.State, tt.state)
			}
			if !got.NextChange.After(now) {
				t.Errorf("next change %s is not after now %s", got.NextChange, now)
			}
		})
	}
}

func TestNextChange(t *testing.T) {
	tests := []struct {
		name     string
		place    place
		now      func(t *testing.T) time.Time
		from, to func(t *testing.T) time.Time // NextChange must fall in [from, to]
	}{
		{
			// v1 told caches to keep "Good Night" for 28.5 hours from here.
			name: "Brasília before dawn changes at today's sunrise", place: brasilia,
			now:  func(t *testing.T) time.Time { return brasilia.at(t, 2026, 11, 24, 1, 0) },
			from: func(t *testing.T) time.Time { return brasilia.at(t, 2026, 11, 24, 5, 0) },
			to:   func(t *testing.T) time.Time { return brasilia.at(t, 2026, 11, 24, 6, 0) },
		},
		{
			name: "Tokyo morning changes at local noon", place: tokyo,
			now:  func(t *testing.T) time.Time { return tokyo.at(t, 2026, 11, 24, 8, 0) },
			from: func(t *testing.T) time.Time { return tokyo.at(t, 2026, 11, 24, 12, 0) },
			to:   func(t *testing.T) time.Time { return tokyo.at(t, 2026, 11, 24, 12, 0) },
		},
		{
			name: "Lisbon noon follows summer time on the change day", place: lisbon,
			now:  func(t *testing.T) time.Time { return lisbon.at(t, 2026, 3, 29, 11, 30) },
			from: func(t *testing.T) time.Time { return lisbon.at(t, 2026, 3, 29, 12, 0) },
			to:   func(t *testing.T) time.Time { return lisbon.at(t, 2026, 3, 29, 12, 0) },
		},
		{
			name: "Tromsø polar night afternoon ends at 18:00", place: tromso,
			now:  func(t *testing.T) time.Time { return tromso.at(t, 2026, 12, 15, 12, 0) },
			from: func(t *testing.T) time.Time { return tromso.at(t, 2026, 12, 15, 18, 0) },
			to:   func(t *testing.T) time.Time { return tromso.at(t, 2026, 12, 15, 18, 0) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compute(tt.now(t), tt.place.loc(t), tt.place.lat, tt.place.lon).NextChange
			if got.Before(tt.from(t)) || got.After(tt.to(t)) {
				t.Errorf("next change = %s, want between %s and %s", got, tt.from(t), tt.to(t))
			}
		})
	}
}

// TestPeriodsChangeAtNextChange checks the invariant behind the ETag and the
// JSON nextChange field, for random moments in many places: the period holds
// until NextChange and changes exactly there.
func TestPeriodsChangeAtNextChange(t *testing.T) {
	places := []place{brasilia, tokyo, sydney, lisbon, tromso, quito, kiritimat, pagoPago, kolkata, kathmandu,
		{"Longyearbyen", 78.22, 15.65, "Arctic/Longyearbyen"},
		{"McMurdo", -77.85, 166.67, "Antarctica/McMurdo"},
		{"Kashgar", 39.47, 75.99, "Asia/Shanghai"}, // clock far from solar time
	}
	rng := rand.New(rand.NewPCG(2026, 11))
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range places {
		loc := p.loc(t)
		for range 300 {
			now := start.Add(time.Duration(rng.Int64N(int64(365 * 24 * time.Hour))))
			r := Compute(now, loc, p.lat, p.lon)
			if !r.NextChange.After(r.Now) {
				t.Fatalf("%s at %s: next change %s is not after now", p.name, r.Now, r.NextChange)
			}
			before := Compute(r.NextChange.Add(-time.Second), loc, p.lat, p.lon)
			after := Compute(r.NextChange, loc, p.lat, p.lon)
			if before.Period != r.Period || after.Period == r.Period {
				t.Fatalf("%s at %s: period %s, one second before next change %s, at next change %s (%s)",
					p.name, r.Now, r.Period, before.Period, after.Period, r.NextChange)
			}
		}
	}
}

func TestProgress(t *testing.T) {
	loc := brasilia.loc(t)
	r := Compute(brasilia.at(t, 2026, 11, 24, 15, 0), loc, brasilia.lat, brasilia.lon)
	p, ok := r.Progress()
	if !ok || p < 0.70 || p > 0.76 {
		t.Errorf("progress at 15:00 = %.3f, %v; want about 0.733", p, ok)
	}
	if _, ok := Compute(brasilia.at(t, 2026, 11, 24, 21, 0), loc, brasilia.lat, brasilia.lon).Progress(); ok {
		t.Error("progress at night should not exist")
	}
	if _, ok := Compute(tromso.at(t, 2026, 12, 15, 12, 0), tromso.loc(t), tromso.lat, tromso.lon).Progress(); ok {
		t.Error("progress in the polar night should not exist")
	}
}

func TestDaylight(t *testing.T) {
	quitoDay := SunOn(quito.lat, quito.lon, quito.loc(t), 2026, 11, 24)
	if d := quitoDay.Daylight(); d < 11*time.Hour+50*time.Minute || d > 12*time.Hour+20*time.Minute {
		t.Errorf("daylight in Quito = %s, want about 12h", d)
	}
	if d := SunOn(tromso.lat, tromso.lon, tromso.loc(t), 2026, 6, 21).Daylight(); d != 24*time.Hour {
		t.Errorf("daylight under the midnight sun = %s, want 24h", d)
	}
	if d := SunOn(tromso.lat, tromso.lon, tromso.loc(t), 2026, 12, 15).Daylight(); d != 0 {
		t.Errorf("daylight in the polar night = %s, want 0", d)
	}
}

func TestTextCoversEveryLanguageAndPeriod(t *testing.T) {
	for _, l := range Langs {
		for _, p := range Periods {
			if Text(l, p) == "" {
				t.Errorf("no greeting for %s in %s", p, l)
			}
		}
	}
}
