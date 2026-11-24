package greeting

// Lang is a language the greetings are written in.
type Lang string

// The supported languages.
const (
	English    Lang = "en"
	Portuguese Lang = "pt-BR"
	Spanish    Lang = "es"
)

// Langs lists the supported languages.
var Langs = []Lang{English, Portuguese, Spanish}

// Periods lists the periods of a day in order.
var Periods = []Period{Morning, Afternoon, Evening}

var texts = map[Lang]map[Period]string{
	English:    {Morning: "Good morning", Afternoon: "Good afternoon", Evening: "Good evening"},
	Portuguese: {Morning: "Bom dia", Afternoon: "Boa tarde", Evening: "Boa noite"},
	Spanish:    {Morning: "Buenos días", Afternoon: "Buenas tardes", Evening: "Buenas noches"},
}

// Text returns the greeting for period p in lang, or "" for an unknown language.
func Text(lang Lang, p Period) string {
	return texts[lang][p]
}
