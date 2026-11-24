// Package card renders a greeting as an SVG card for READMEs.
//
// Every option is a closed choice and no text from the request reaches the
// card: the words come from fixed tables, the numbers from the computed
// greeting. That is what lets the tests render every combination.
package card

import (
	"embed"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/ruanpato/polite-greetings/internal/greeting"
)

// Layout is the shape of the card.
type Layout string

// The layouts.
const (
	Compact Layout = "compact" // icon and greeting in one line
	Classic Layout = "classic" // icon, greeting and the chosen facts
	Sky     Layout = "sky"     // greeting, local time and the sun's path
)

// Fact is a piece of information the card can show besides the greeting.
type Fact string

// The facts, in the order the card always shows them.
const (
	FactSun       Fact = "sun"
	FactDaylength Fact = "daylength"
	FactTime      Fact = "time"
	FactTZ        Fact = "tz"
)

// Clock is how times are written.
type Clock string

// The clocks.
const (
	Clock24 Clock = "24h"
	Clock12 Clock = "12h"
)

// Options are the rendering choices. Start from Defaults.
type Options struct {
	Layout  Layout
	Facts   []Fact // shown in the fixed order whatever the order here; ignored by Compact
	Theme   Theme
	Colors  Colors // six hex digits each; an empty field keeps the theme's color
	Radius  *int   // nil keeps the layout's default
	Border  bool
	Animate bool
	Clock   Clock
	Width   int // display width in pixels; 0 keeps the layout's natural width
	Lang    greeting.Lang
	Zone    string // IANA name of the place's time zone, for FactTZ
}

// Defaults returns the options the contract documents as defaults.
func Defaults() Options {
	return Options{Layout: Classic, Facts: []Fact{FactSun}, Theme: Auto, Border: true, Clock: Clock24, Lang: greeting.English}
}

// Render writes the card for the greeting r.
func Render(w io.Writer, r greeting.Result, o Options) error {
	v, err := newView(r, o)
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, "card", v)
}

//go:embed card.svg.tmpl
var templates embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{"x": escape}).ParseFS(templates, "card.svg.tmpl"))

// escape makes text safe inside XML.
func escape(s string) (string, error) {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Natural sizes of the layouts, in card units.
const (
	compactW, compactH = 240, 48
	classicW, classicH = 440, 100
	skyW, skyH         = 440, 190
	lineHeight         = 20
)

// Text boxes used to wrap and to check that texts fit, in card units.
const (
	classicTextX, classicTextRight = 96, 424
	classicFactSize                = 14
	regularWidth, boldWidth        = 0.58, 0.62 // average glyph width per em, on the safe side
)

// view is everything the template needs, already computed and formatted.
type view struct {
	ID            string
	Lang          string
	Title         string
	Layout        Layout
	Period        greeting.Period
	Width, Height int // display size
	W, H          int // viewBox size
	RectW, RectH  int
	Radius        string
	Styles        []style // the light or only palette first; the dark one under a media query when Auto
	Auto          bool
	Animate       bool
	Greeting      string
	GreetingY     int
	Lines         []line // Classic
	Sky           skyView
}

type line struct {
	Y    int
	Text string
}

type style struct {
	ID     string
	P      Palette
	Border bool
}

type skyView struct {
	Day, MidnightSun, Ends    bool
	ArcFull, ArcDone, ArcLeft string
	Fill, SunX, SunY          string
	Rise, Set, Center         string
	Time, Zone                string
}

func newView(r greeting.Result, o Options) (view, error) {
	l, ok := labelsFor[o.Lang]
	if !ok {
		return view{}, fmt.Errorf("card: unknown language %q", o.Lang)
	}
	if o.Clock != Clock24 && o.Clock != Clock12 {
		return view{}, fmt.Errorf("card: unknown clock %q", o.Clock)
	}
	styles, err := stylesFor(o)
	if err != nil {
		return view{}, err
	}
	v := view{
		ID:       "pg",
		Lang:     string(o.Lang),
		Layout:   o.Layout,
		Period:   r.Period,
		Styles:   styles,
		Auto:     o.Theme == Auto,
		Animate:  o.Animate,
		Greeting: greeting.Text(o.Lang, r.Period),
	}
	facts := texts(r, o, l)
	v.Title = strings.Join(append([]string{v.Greeting}, facts...), ". ") + "."

	radius := 12
	switch o.Layout {
	case Compact:
		v.W, v.H, radius = compactW, compactH, 24
	case Classic:
		v.W, v.H, v.GreetingY = classicW, classicH, 59
		lines := wrap(facts, int(math.Floor((classicTextRight-classicTextX)/(classicFactSize*regularWidth))))
		for i, t := range lines {
			v.Lines = append(v.Lines, line{Y: 76 + lineHeight*i, Text: t})
		}
		if len(lines) > 0 {
			v.GreetingY = 50
			v.H += lineHeight * (len(lines) - 1)
		}
	case Sky:
		v.W, v.H = skyW, skyH
		v.Sky = newSky(r, o, l)
	default:
		return view{}, fmt.Errorf("card: unknown layout %q", o.Layout)
	}
	v.RectW, v.RectH = v.W-1, v.H-1
	if o.Radius != nil {
		radius = *o.Radius
	}
	v.Radius = num(math.Min(float64(radius), float64(v.H)/2))
	v.Width, v.Height = v.W, v.H
	if o.Width > 0 {
		v.Width = o.Width
		v.Height = int(math.Round(float64(o.Width) * float64(v.H) / float64(v.W)))
	}
	return v, nil
}

// texts returns the chosen facts as short texts, in the fixed order. Sunrise
// and sunset are separate texts so that a line can break between them.
func texts(r greeting.Result, o Options, l labels) []string {
	if o.Layout == Compact {
		return nil
	}
	has := func(f Fact) bool { return slices.Contains(o.Facts, f) }
	day := r.Today
	var out []string
	if has(FactSun) {
		switch day.State {
		case greeting.SunNormal:
			out = append(out, l.Sunrise+" "+clock(day.Sunrise.Round(time.Minute), o.Clock), l.Sunset+" "+clock(day.Sunset.Round(time.Minute), o.Clock))
		case greeting.PolarNight:
			out = append(out, l.NoSunrise)
		case greeting.MidnightSun:
			out = append(out, l.NoSunset)
		}
	}
	if has(FactDaylength) && day.State == greeting.SunNormal {
		out = append(out, l.daylight(day.Daylight()))
	}
	if has(FactTime) {
		out = append(out, fmt.Sprintf(l.LocalTime, clock(r.Now, o.Clock)))
	}
	if has(FactTZ) {
		out = append(out, zoneLabel(o.Zone, r.Now))
	}
	return out
}

func newSky(r greeting.Result, o Options, l labels) skyView {
	has := func(f Fact) bool { return slices.Contains(o.Facts, f) }
	day := r.Today
	s := skyView{ArcFull: arc(0, 1), Ends: day.State == greeting.SunNormal, MidnightSun: day.State == greeting.MidnightSun}
	if p, ok := r.Progress(); ok {
		x, y := skyPoint(p)
		s.Day = true
		s.ArcDone, s.ArcLeft = arc(0, p), arc(p, 1)
		s.Fill = s.ArcDone + " L" + num(x) + " " + num(skyHorizon) + "Z"
		s.SunX, s.SunY = num(x), num(y)
	}
	switch {
	case has(FactSun) && day.State == greeting.SunNormal:
		s.Rise, s.Set = clock(day.Sunrise.Round(time.Minute), o.Clock), clock(day.Sunset.Round(time.Minute), o.Clock)
	case has(FactSun) && day.State == greeting.PolarNight:
		s.Center = l.NoSunrise
	case has(FactSun) && day.State == greeting.MidnightSun:
		s.Center = l.NoSunset
	}
	if s.Center == "" && has(FactDaylength) && day.State == greeting.SunNormal {
		s.Center = l.daylight(day.Daylight())
	}
	if has(FactTime) {
		s.Time = clock(r.Now, o.Clock)
	}
	if has(FactTZ) {
		s.Zone = zoneLabel(o.Zone, r.Now)
	}
	return s
}

// The sun's path in the sky layout: half an ellipse over the horizon line.
const (
	skyHorizon   = 150.0
	skyCX        = 220.0
	skyRX, skyRY = 180.0, 62.0
)

// skyPoint is where the sun is on the path when share f of the daylight has passed.
func skyPoint(f float64) (x, y float64) {
	return skyCX - skyRX*math.Cos(math.Pi*f), skyHorizon - skyRY*math.Sin(math.Pi*f)
}

// arc is the SVG path of the sun's path between shares f0 and f1 of the daylight.
func arc(f0, f1 float64) string {
	x0, y0 := skyPoint(f0)
	x1, y1 := skyPoint(f1)
	return fmt.Sprintf("M%s %s A%s %s 0 0 1 %s %s", num(x0), num(y0), num(skyRX), num(skyRY), num(x1), num(y1))
}

// wrap joins texts with " · " into lines of at most maxChars characters,
// breaking only between texts.
func wrap(texts []string, maxChars int) []string {
	var lines []string
	cur := ""
	for _, t := range texts {
		next := t
		if cur != "" {
			next = cur + " · " + t
		}
		if cur != "" && utf8.RuneCountInString(next) > maxChars {
			lines = append(lines, cur)
			next = t
		}
		cur = next
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func clock(t time.Time, c Clock) string {
	if c == Clock12 {
		return t.Format("3:04 PM")
	}
	return t.Format("15:04")
}

// zoneLabel names the place's time zone by the last part of its IANA name,
// with the UTC offset: "Sao Paulo · UTC-03:00".
func zoneLabel(zone string, now time.Time) string {
	offset := "UTC" + now.Format("-07:00")
	name := strings.ReplaceAll(zone[strings.LastIndex(zone, "/")+1:], "_", " ")
	if name == "" || strings.HasPrefix(name, "UTC") || strings.HasPrefix(name, "GMT") {
		return offset
	}
	return name + " · " + offset
}

// num formats a coordinate with at most two decimals.
func num(v float64) string {
	return strconv.FormatFloat(math.Round(v*100)/100+0, 'f', -1, 64)
}
