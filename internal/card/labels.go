package card

import (
	"fmt"
	"time"

	"github.com/ruanpato/polite-greetings/internal/greeting"
)

// labels are the words of a card in one language.
type labels struct {
	Sunrise, Sunset     string
	DaylightFormat      string // hours, minutes
	LocalTime           string // the formatted time
	NoSunrise, NoSunset string
}

var labelsFor = map[greeting.Lang]labels{
	greeting.English: {
		Sunrise: "Sunrise", Sunset: "Sunset",
		DaylightFormat: "%dh %02dm of daylight", LocalTime: "Local time %s",
		NoSunrise: "No sunrise today", NoSunset: "No sunset today",
	},
	greeting.Portuguese: {
		Sunrise: "Nascer do sol", Sunset: "Pôr do sol",
		DaylightFormat: "%d h %02d min de luz", LocalTime: "Hora local %s",
		NoSunrise: "Sem nascer do sol hoje", NoSunset: "Sem pôr do sol hoje",
	},
	greeting.Spanish: {
		Sunrise: "Amanecer", Sunset: "Atardecer",
		DaylightFormat: "%d h %02d min de luz", LocalTime: "Hora local %s",
		NoSunrise: "Hoy no sale el sol", NoSunset: "Hoy no se pone el sol",
	},
}

func (l labels) daylight(d time.Duration) string {
	d = d.Round(time.Minute)
	return fmt.Sprintf(l.DaylightFormat, int(d/time.Hour), int(d%time.Hour/time.Minute))
}
