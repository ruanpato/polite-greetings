package card

import (
	"math"
	"strconv"
	"strings"

	"github.com/ruanpato/polite-greetings/internal/astro"
	"github.com/ruanpato/polite-greetings/internal/greeting"
)

// The cosmos layout draws the Earth from space, centered on the place's
// longitude with the equator level, so the place's latitude is a straight
// band. Around it, the sun sits on an orbit in the direction its light comes
// from, and the moon on an inner orbit, as far from the sun as its phase.
const (
	cosmosW, cosmosH             = 440, 200
	globeX, globeY, globeR       = 102.0, 100.0, 60.0
	sunOrbit, moonOrbit          = 90.0, 73.0
	moonRadius                   = 6.5
	bandHalfWidth                = 3.0 // degrees of latitude on each side of the place
	cosmosTextX, cosmosTextRight = 206, 424
	cosmosFactSize               = 13
	cosmosLineHeight             = 19
	cosmosFirstLine              = 86
)

const deg = math.Pi / 180

type cosmosView struct {
	Dots, Grid, Night     string // SVG path data
	BandY, BandH, LatY    string
	SunX, SunY            string
	MoonX, MoonY, MoonLit string
	Lines                 []line
}

func newCosmos(r greeting.Result, o Options, facts []string) cosmosView {
	var c cosmosView
	for i, t := range wrap(facts, int(math.Floor((cosmosTextRight-cosmosTextX)/(cosmosFactSize*regularWidth)))) {
		c.Lines = append(c.Lines, line{Y: cosmosFirstLine + cosmosLineHeight*i, Text: t})
	}

	// Land: the points facing us.
	var dots strings.Builder
	for _, p := range land {
		x, y, z := toView(float64(p[0])/10, float64(p[1])/10, o.Lon)
		if z > 0 {
			dots.WriteString("M" + num1(globeX+globeR*x) + " " + num1(globeY-globeR*y) + "h0")
		}
	}
	c.Dots = dots.String()

	// Meridians every 30° (the visible halves of ellipses) and parallels every
	// 30° up to 60° (straight lines, since the equator is level).
	var grid strings.Builder
	for m := -180.0; m < 180; m += 30 {
		d := astro.Wrap180(m - o.Lon)
		if math.Abs(d) >= 90 {
			continue
		}
		rx := math.Abs(globeR * math.Sin(d*deg))
		top, bottom := num1(globeY-globeR), num1(globeY+globeR)
		if rx < 0.5 {
			grid.WriteString("M" + num1(globeX) + " " + top + "V" + bottom)
			continue
		}
		sweep := "0"
		if d > 0 {
			sweep = "1"
		}
		grid.WriteString("M" + num1(globeX) + " " + top + "A" + num1(rx) + " " + num1(globeR) + " 0 0 " + sweep + " " + num1(globeX) + " " + bottom)
	}
	for p := -60.0; p <= 60; p += 30 {
		y, half := globeY-globeR*math.Sin(p*deg), globeR*math.Cos(p*deg)
		grid.WriteString("M" + num1(globeX-half) + " " + num1(y) + "H" + num1(globeX+half))
	}
	c.Grid = grid.String()

	// Night: the side of the globe away from the sun.
	sunLat, sunLon := astro.Subsolar(r.Now)
	sx, sy, sz := toView(sunLat, sunLon, o.Lon)
	light := math.Atan2(-sy, sx) // screen angle the light comes from
	c.Night = litPath(globeX, globeY, globeR, 1-(1+sz)/2, light+math.Pi, 24)

	// The place's latitude, as a band and a line.
	top := globeY - globeR*math.Sin(math.Min(o.Lat+bandHalfWidth, 90)*deg)
	bottom := globeY - globeR*math.Sin(math.Max(o.Lat-bandHalfWidth, -90)*deg)
	c.BandY, c.BandH = num1(top), num1(bottom-top)
	c.LatY = num1(globeY - globeR*math.Sin(o.Lat*deg))

	// The sun on its orbit, toward the light; the moon on the inner orbit,
	// turned from the sun by its age in the cycle (opposite at full moon).
	c.SunX, c.SunY = num1(globeX+sunOrbit*math.Cos(light)), num1(globeY+sunOrbit*math.Sin(light))
	m := astro.MoonAt(r.Now)
	angle := light - m.Age*2*math.Pi
	mx, my := globeX+moonOrbit*math.Cos(angle), globeY+moonOrbit*math.Sin(angle)
	c.MoonX, c.MoonY = num1(mx), num1(my)
	c.MoonLit = litPath(mx, my, moonRadius, m.Illumination, moonLitSide(m, o.Lat), 16)
	return c
}

// toView turns a point on the Earth into view coordinates for a globe centered
// on longitude lon0 with the equator level: x to the right, y up, z toward us.
func toView(lat, lon, lon0 float64) (x, y, z float64) {
	d, p := (lon-lon0)*deg, lat*deg
	return math.Cos(p) * math.Sin(d), math.Sin(p), math.Cos(p) * math.Cos(d)
}

// moonLitSide is the screen angle of the moon's lit side as seen from the
// place: on the right while waxing in the northern hemisphere, mirrored in the
// southern one.
func moonLitSide(m astro.Moon, lat float64) float64 {
	if m.Waxing != (lat < 0) {
		return 0
	}
	return math.Pi
}

// litPath outlines the lit part of a sphere seen as a disk of radius r at
// (cx, cy): share k of the disk is lit, on the side the light comes from, at
// screen angle angle (0 is right, π/2 is down). The outline is half the rim and
// the terminator, an ellipse, each drawn with steps segments.
func litPath(cx, cy, r, k, angle float64, steps int) string {
	ca, sa := math.Cos(angle), math.Sin(angle)
	var b strings.Builder
	point := func(x, y float64) {
		if b.Len() == 0 {
			b.WriteString("M")
		} else {
			b.WriteString("L")
		}
		b.WriteString(num1(cx + x*ca - y*sa))
		b.WriteString(" ")
		b.WriteString(num1(cy + x*sa + y*ca))
	}
	for i := 0; i <= steps; i++ {
		t := -math.Pi/2 + math.Pi*float64(i)/float64(steps)
		point(r*math.Cos(t), r*math.Sin(t))
	}
	rx, sign := r*math.Abs(2*k-1), 1.0
	if k >= 0.5 {
		sign = -1 // gibbous: the terminator bulges into the dark side
	}
	for i := steps; i >= 0; i-- {
		t := -math.Pi/2 + math.Pi*float64(i)/float64(steps)
		point(sign*rx*math.Cos(t), r*math.Sin(t))
	}
	b.WriteString("Z")
	return b.String()
}

// num1 formats a coordinate with at most one decimal.
func num1(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10+0, 'f', -1, 64)
}
