package card

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Theme picks the card's colors.
type Theme string

// The themes.
const (
	Auto  Theme = "auto" // light, or dark when the viewer prefers dark
	Light Theme = "light"
	Dark  Theme = "dark"
)

// Palette holds a card's colors as six hex digits.
type Palette struct {
	Bg     string // background
	Accent string // greeting, icon and sun
	Text   string // facts and labels; the border uses it at 25% opacity
}

// Colors overrides parts of a theme. Empty fields keep the theme's color.
type Colors = Palette

// Palettes are the themes' colors. The dark one uses the project's colors; the
// light one needs a darker green to stay readable on white.
var Palettes = map[Theme]Palette{
	Light: {Bg: "FFFFFF", Accent: "007A5A", Text: "474555"},
	Dark:  {Bg: "474555", Accent: "00C894", Text: "C8FCEA"},
}

var hexColor = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

// with returns p with the colors set in c.
func (p Palette) with(c Colors) Palette {
	for _, f := range []struct {
		dst *string
		src string
	}{{&p.Bg, c.Bg}, {&p.Accent, c.Accent}, {&p.Text, c.Text}} {
		if f.src != "" {
			*f.dst = strings.ToUpper(f.src)
		}
	}
	return p
}

func stylesFor(o Options) ([]style, error) {
	for _, c := range []string{o.Colors.Bg, o.Colors.Accent, o.Colors.Text} {
		if c != "" && !hexColor.MatchString(c) {
			return nil, fmt.Errorf("card: color %q is not six hex digits", c)
		}
	}
	mk := func(t Theme) style { return style{ID: "pg", P: Palettes[t].with(o.Colors), Border: o.Border} }
	switch o.Theme {
	case Auto:
		return []style{mk(Light), mk(Dark)}, nil
	case Light, Dark:
		return []style{mk(o.Theme)}, nil
	default:
		return nil, fmt.Errorf("card: unknown theme %q", o.Theme)
	}
}

// Contrast is the WCAG 2 contrast ratio between two colors given as six hex
// digits, from 1 to 21.
func Contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// luminance is the WCAG 2 relative luminance of a color given as six hex digits.
func luminance(hex string) float64 {
	var rgb [3]float64
	for i := range rgb {
		v, _ := strconv.ParseUint(hex[2*i:2*i+2], 16, 8)
		c := float64(v) / 255
		if c <= 0.04045 {
			rgb[i] = c / 12.92
		} else {
			rgb[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}
