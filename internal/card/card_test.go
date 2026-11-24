package card

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ruanpato/polite-greetings/internal/greeting"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// moment is a fixed greeting to render: a place and a local wall-clock time.
type moment struct {
	name     string
	zone     string
	lat, lon float64
	at       [5]int // year, month, day, hour, minute
}

var (
	morning     = moment{"morning", "America/Sao_Paulo", -15.78, -47.93, [5]int{2026, 11, 24, 9, 0}}
	afternoon   = moment{"afternoon", "America/Sao_Paulo", -15.78, -47.93, [5]int{2026, 11, 24, 15, 0}}
	evening     = moment{"evening", "America/Sao_Paulo", -15.78, -47.93, [5]int{2026, 11, 24, 21, 4}}
	polarNight  = moment{"polar-night", "Europe/Oslo", 69.65, 18.96, [5]int{2026, 12, 15, 12, 0}}
	midnightSun = moment{"midnight-sun", "Europe/Oslo", 69.65, 18.96, [5]int{2026, 6, 21, 1, 0}}
	longZone    = moment{"long-zone", "America/Argentina/ComodRivadavia", -45.87, -67.5, [5]int{2026, 11, 24, 15, 0}}
	moments     = []moment{morning, afternoon, evening, polarNight, midnightSun, longZone}
)

func (m moment) result(t *testing.T) greeting.Result {
	t.Helper()
	loc, err := time.LoadLocation(m.zone)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(m.at[0], time.Month(m.at[1]), m.at[2], m.at[3], m.at[4], 0, 0, loc)
	return greeting.Compute(now, loc, m.lat, m.lon)
}

func opts(edit func(*Options)) Options {
	o := Defaults()
	edit(&o)
	return o
}

func intp(v int) *int { return &v }

func TestGolden(t *testing.T) {
	cases := []struct {
		name string
		m    moment
		o    Options
	}{
		{"compact-en-morning-light", morning, opts(func(o *Options) { o.Layout, o.Theme = Compact, Light })},
		{"compact-pt-BR-evening-custom", evening, opts(func(o *Options) {
			o.Layout, o.Theme, o.Lang, o.Border = Compact, Dark, greeting.Portuguese, false
			o.Colors = Colors{Bg: "0D1117", Accent: "F78166", Text: "C9D1D9"}
		})},
		{"classic-pt-BR-afternoon-dark", afternoon, opts(func(o *Options) { o.Theme, o.Lang = Dark, greeting.Portuguese })},
		{"classic-en-evening-all-facts-12h-light", evening, opts(func(o *Options) {
			o.Theme, o.Clock, o.Facts = Light, Clock12, []Fact{FactTZ, FactTime, FactDaylength, FactSun}
		})},
		{"classic-en-midnight-sun-auto", midnightSun, opts(func(o *Options) { o.Facts = []Fact{FactSun, FactDaylength} })},
		{"classic-en-morning-width-600-square", morning, opts(func(o *Options) {
			o.Width, o.Radius = 600, intp(0)
		})},
		{"sky-es-afternoon-auto", afternoon, opts(func(o *Options) {
			o.Layout, o.Lang, o.Facts = Sky, greeting.Spanish, []Fact{FactSun, FactDaylength, FactTime}
		})},
		{"sky-pt-BR-evening-dark-animated", evening, opts(func(o *Options) {
			o.Layout, o.Theme, o.Lang, o.Animate = Sky, Dark, greeting.Portuguese, true
			o.Facts = []Fact{FactSun, FactDaylength, FactTime, FactTZ}
		})},
		{"sky-en-polar-night-light", polarNight, opts(func(o *Options) {
			o.Layout, o.Theme, o.Facts = Sky, Light, []Fact{FactSun, FactTime, FactTZ}
		})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.o.Zone = c.m.zone
			var buf bytes.Buffer
			if err := Render(&buf, c.m.result(t), c.o); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", c.name+".svg")
			if *update {
				if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run `go test ./internal/card -update` to create it)", err)
			}
			if !bytes.Equal(buf.Bytes(), want) {
				t.Errorf("%s changed; run `go test ./internal/card -update` and review the diff", path)
			}
		})
	}
}

// TestEveryCombination renders every combination of the closed options for
// several moments, and checks that each card is well-formed XML whose texts fit.
func TestEveryCombination(t *testing.T) {
	all := []Fact{FactSun, FactDaylength, FactTime, FactTZ}
	var factSets [][]Fact
	for mask := range 1 << len(all) {
		set := []Fact{}
		for i, f := range all {
			if mask&(1<<i) != 0 {
				set = append(set, f)
			}
		}
		factSets = append(factSets, set)
	}
	results := map[string]greeting.Result{}
	for _, m := range moments {
		results[m.name] = m.result(t)
	}
	n := 0
	for _, m := range moments {
		for _, layout := range []Layout{Compact, Classic, Sky} {
			for _, theme := range []Theme{Auto, Light, Dark} {
				for _, facts := range factSets {
					for _, lang := range greeting.Langs {
						for _, clk := range []Clock{Clock24, Clock12} {
							for _, flags := range []struct{ animate, border bool }{{false, true}, {true, false}} {
								o := Options{Layout: layout, Theme: theme, Facts: facts, Lang: lang, Clock: clk,
									Animate: flags.animate, Border: flags.border, Zone: m.zone}
								v, err := newView(results[m.name], o)
								if err != nil {
									t.Fatalf("%s %+v: %v", m.name, o, err)
								}
								for _, problem := range overflows(v) {
									t.Errorf("%s %s %s %v %s %s: %s", m.name, layout, theme, facts, lang, clk, problem)
								}
								var buf bytes.Buffer
								if err := tmpl.ExecuteTemplate(&buf, "card", v); err != nil {
									t.Fatal(err)
								}
								if err := wellFormed(buf.Bytes()); err != nil {
									t.Fatalf("%s %+v: not well-formed XML: %v", m.name, o, err)
								}
								n++
							}
						}
					}
				}
			}
		}
	}
	t.Logf("rendered %d cards", n)
}

func wellFormed(b []byte) error {
	d := xml.NewDecoder(bytes.NewReader(b))
	for {
		if _, err := d.Token(); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// width estimates how wide a text renders, on the safe side.
func width(s string, size float64, bold bool) float64 {
	f := regularWidth
	if bold {
		f = boldWidth
	}
	return float64(utf8.RuneCountInString(s)) * size * f
}

// overflows lists the texts of a card that may not fit their place.
func overflows(v view) []string {
	var out []string
	check := func(ok bool, format string, args ...any) {
		if !ok {
			out = append(out, fmt.Sprintf(format, args...))
		}
	}
	switch v.Layout {
	case Compact:
		check(46+width(v.Greeting, 18, true) <= compactW-12, "greeting %q", v.Greeting)
	case Classic:
		check(classicTextX+width(v.Greeting, 28, true) <= classicTextRight, "greeting %q", v.Greeting)
		for _, l := range v.Lines {
			check(classicTextX+width(l.Text, classicFactSize, false) <= classicTextRight, "line %q", l.Text)
		}
	case Sky:
		s := v.Sky
		greetEnd := 78 + width(v.Greeting, 24, true)
		if s.Time != "" {
			check(greetEnd+12 <= 416-width(s.Time, 20, true), "greeting %q against time %q", v.Greeting, s.Time)
		} else {
			check(greetEnd <= 424, "greeting %q", v.Greeting)
		}
		check(width(s.Zone, 11, false) <= 416-78, "zone %q", s.Zone)
		left, right := 40.0, 400.0
		if s.Rise != "" {
			left, right = 40+width(s.Rise, 12, false)+8, 400-width(s.Set, 12, false)-8
		}
		if s.Center != "" {
			half := width(s.Center, 12, false) / 2
			check(220-half >= left && 220+half <= right, "center %q between %q and %q", s.Center, s.Rise, s.Set)
		}
	}
	return out
}

func TestDefaultPalettesAreReadable(t *testing.T) {
	for theme, p := range Palettes {
		// The greeting is large text (WCAG: 3:1); the facts are normal text (4.5:1).
		if c := Contrast(p.Accent, p.Bg); c < 3 {
			t.Errorf("%s: accent on background is %.2f:1, want at least 3:1", theme, c)
		}
		if c := Contrast(p.Text, p.Bg); c < 4.5 {
			t.Errorf("%s: text on background is %.2f:1, want at least 4.5:1", theme, c)
		}
	}
	if c := Contrast("000000", "FFFFFF"); c < 20.99 || c > 21.01 {
		t.Errorf("black on white = %.2f:1, want 21:1", c)
	}
}

func TestLabelsCoverEveryLanguage(t *testing.T) {
	for _, l := range greeting.Langs {
		lb, ok := labelsFor[l]
		if !ok || lb.Sunrise == "" || lb.Sunset == "" || lb.DaylightFormat == "" || lb.LocalTime == "" || lb.NoSunrise == "" || lb.NoSunset == "" {
			t.Errorf("missing labels for %s", l)
		}
	}
}

func TestFactsKeepTheirOrder(t *testing.T) {
	r := evening.result(t)
	a, _ := newView(r, opts(func(o *Options) { o.Facts, o.Zone = []Fact{FactTZ, FactSun}, evening.zone }))
	b, _ := newView(r, opts(func(o *Options) { o.Facts, o.Zone = []Fact{FactSun, FactTZ}, evening.zone }))
	if a.Title != b.Title {
		t.Errorf("order of facts changed the card: %q vs %q", a.Title, b.Title)
	}
}

func TestRejectsUnknownOptions(t *testing.T) {
	r := morning.result(t)
	bad := []Options{
		opts(func(o *Options) { o.Layout = "poster" }),
		opts(func(o *Options) { o.Theme = "neon" }),
		opts(func(o *Options) { o.Clock = "36h" }),
		opts(func(o *Options) { o.Lang = "fr" }),
		opts(func(o *Options) { o.Colors.Bg = "#FFFFFF" }),
		opts(func(o *Options) { o.Colors.Text = "red" }),
	}
	for _, o := range bad {
		if err := Render(io.Discard, r, o); err == nil {
			t.Errorf("Render accepted %+v", o)
		}
	}
}

func TestWrap(t *testing.T) {
	got := wrap([]string{"aaaa", "bbbb", "cccc"}, 11)
	want := []string{"aaaa · bbbb", "cccc"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("wrap = %q, want %q", got, want)
	}
	if got := wrap(nil, 10); got != nil {
		t.Errorf("wrap(nil) = %q, want nil", got)
	}
}

func TestZoneLabel(t *testing.T) {
	now := time.Date(2026, 11, 24, 21, 4, 0, 0, time.FixedZone("", -3*3600))
	for zone, want := range map[string]string{
		"America/Sao_Paulo":              "Sao Paulo · UTC-03:00",
		"America/Argentina/Buenos_Aires": "Buenos Aires · UTC-03:00",
		"UTC":                            "UTC-03:00",
		"Etc/GMT+3":                      "UTC-03:00",
	} {
		if got := zoneLabel(zone, now); got != want {
			t.Errorf("zoneLabel(%q) = %q, want %q", zone, got, want)
		}
	}
}
