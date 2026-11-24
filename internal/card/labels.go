package card

import (
	"fmt"
	"time"

	"github.com/ruanpato/polite-greetings/internal/astro"
	"github.com/ruanpato/polite-greetings/internal/greeting"
)

// labels are the words of a card in one language.
type labels struct {
	Sunrise, Sunset     string
	DaylightFormat      string // hours, minutes
	LocalTime           string // the formatted time
	NoSunrise, NoSunset string
	MoonFormat          string // phase name, percent lit
	Phases              map[astro.Phase]string
}

var labelsFor = map[greeting.Lang]labels{
	greeting.English: {
		Sunrise: "Sunrise", Sunset: "Sunset",
		DaylightFormat: "%dh %02dm of daylight", LocalTime: "Local time %s",
		NoSunrise: "No sunrise today", NoSunset: "No sunset today",
		MoonFormat: "%s · %d%%",
		Phases: map[astro.Phase]string{
			astro.NewMoon: "New moon", astro.WaxingCrescent: "Waxing crescent moon",
			astro.FirstQuarter: "First quarter moon", astro.WaxingGibbous: "Waxing gibbous moon",
			astro.FullMoon: "Full moon", astro.WaningGibbous: "Waning gibbous moon",
			astro.LastQuarter: "Last quarter moon", astro.WaningCrescent: "Waning crescent moon",
		},
	},
	greeting.Portuguese: {
		Sunrise: "Nascer do sol", Sunset: "Pôr do sol",
		DaylightFormat: "%d h %02d min de luz", LocalTime: "Hora local %s",
		NoSunrise: "Sem nascer do sol hoje", NoSunset: "Sem pôr do sol hoje",
		MoonFormat: "%s · %d%%",
		Phases: map[astro.Phase]string{
			astro.NewMoon: "Lua nova", astro.WaxingCrescent: "Lua crescente",
			astro.FirstQuarter: "Quarto crescente", astro.WaxingGibbous: "Lua gibosa crescente",
			astro.FullMoon: "Lua cheia", astro.WaningGibbous: "Lua gibosa minguante",
			astro.LastQuarter: "Quarto minguante", astro.WaningCrescent: "Lua minguante",
		},
	},
	greeting.Spanish: {
		Sunrise: "Amanecer", Sunset: "Atardecer",
		DaylightFormat: "%d h %02d min de luz", LocalTime: "Hora local %s",
		NoSunrise: "Hoy no sale el sol", NoSunset: "Hoy no se pone el sol",
		MoonFormat: "%s · %d%%",
		Phases: map[astro.Phase]string{
			astro.NewMoon: "Luna nueva", astro.WaxingCrescent: "Luna creciente",
			astro.FirstQuarter: "Cuarto creciente", astro.WaxingGibbous: "Luna gibosa creciente",
			astro.FullMoon: "Luna llena", astro.WaningGibbous: "Luna gibosa menguante",
			astro.LastQuarter: "Cuarto menguante", astro.WaningCrescent: "Luna menguante",
		},
	},
}

func (l labels) daylight(d time.Duration) string {
	d = d.Round(time.Minute)
	return fmt.Sprintf(l.DaylightFormat, int(d/time.Hour), int(d%time.Hour/time.Minute))
}
