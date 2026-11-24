// Package httpapi serves the API described in api/openapi.yaml, plus the page.
//
// Requests are checked against the contract before they reach Server, so
// Server only checks what the contract cannot say (that a time zone exists).
package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ruanpato/polite-greetings/api"
	"github.com/ruanpato/polite-greetings/internal/astro"
	"github.com/ruanpato/polite-greetings/internal/card"
	"github.com/ruanpato/polite-greetings/internal/greeting"
	"github.com/ruanpato/polite-greetings/internal/oapi"
)

// cacheControl makes caches, GitHub's image proxy among them, check back with
// If-None-Match before reusing a response.
const cacheControl = "no-cache"

// Server implements the operations of the contract.
type Server struct {
	Now       func() time.Time // the clock; tests fix it
	Version   string           // the contract's info.version
	Commit    string           // the Git commit of the build
	Readiness *Readiness
}

var _ oapi.StrictServerInterface = (*Server)(nil)

// GetGreeting returns the greeting for a place, as an SVG card, text or JSON.
func (s *Server) GetGreeting(_ context.Context, req oapi.GetGreetingRequestObject) (oapi.GetGreetingResponseObject, error) {
	p := req.Params
	loc, err := loadLocation(p.Tz)
	if err != nil {
		return badRequest(oapi.ParameterError{Parameter: "tz", Detail: err.Error()}), nil
	}
	r := greeting.Compute(s.Now(), loc, p.Lat, p.Lon)
	lang := greeting.English
	if p.Lang != nil {
		lang = greeting.Lang(*p.Lang)
	}

	format := oapi.Svg
	if p.Format != nil {
		format = *p.Format
	}
	var body []byte
	var asJSON oapi.Greeting
	switch format {
	case oapi.Txt:
		body = []byte(greeting.Text(lang, r.Period))
	case oapi.Json:
		asJSON = greetingJSON(r, lang, p.Tz)
		if body, err = json.Marshal(asJSON); err != nil {
			return nil, err
		}
	default:
		var buf bytes.Buffer
		if err := card.Render(&buf, r, cardOptions(p, lang)); err != nil {
			return nil, err
		}
		body = buf.Bytes()
	}

	etag := etagOf(body)
	cc := cacheControl
	if matches(p.IfNoneMatch, etag) {
		return oapi.GetGreeting304Response{Headers: oapi.GetGreeting304ResponseHeaders{ETag: &etag, CacheControl: &cc}}, nil
	}
	headers := oapi.GetGreeting200ResponseHeaders{ETag: &etag, CacheControl: &cc}
	switch format {
	case oapi.Txt:
		return oapi.GetGreeting200TextplainCharsetUtf8Response{Body: bytes.NewReader(body), ContentLength: int64(len(body)), Headers: headers}, nil
	case oapi.Json:
		return oapi.GetGreeting200JSONResponse{Body: asJSON, Headers: headers}, nil
	default:
		return oapi.GetGreeting200ImagesvgXmlCharsetUtf8Response{Body: bytes.NewReader(body), ContentLength: int64(len(body)), Headers: headers}, nil
	}
}

// GetHealth reports that the process runs, with the contract version and the
// commit of the build.
func (s *Server) GetHealth(context.Context, oapi.GetHealthRequestObject) (oapi.GetHealthResponseObject, error) {
	return oapi.GetHealth200JSONResponse(s.health(oapi.Ok)), nil
}

// GetReadiness reports whether the service takes traffic: 503 once it has
// started shutting down.
func (s *Server) GetReadiness(context.Context, oapi.GetReadinessRequestObject) (oapi.GetReadinessResponseObject, error) {
	if !s.Readiness.ready() {
		return oapi.GetReadiness503JSONResponse(s.health(oapi.ShuttingDown)), nil
	}
	return oapi.GetReadiness200JSONResponse(s.health(oapi.Ok)), nil
}

func (s *Server) health(status oapi.HealthStatus) oapi.Health {
	commit := s.Commit
	if commit == "" {
		commit = "unknown"
	}
	return oapi.Health{Status: status, Version: s.Version, Commit: commit}
}

// GetOpenAPIDocument serves the contract.
func (s *Server) GetOpenAPIDocument(context.Context, oapi.GetOpenAPIDocumentRequestObject) (oapi.GetOpenAPIDocumentResponseObject, error) {
	return oapi.GetOpenAPIDocument200ApplicationyamlResponse{Body: bytes.NewReader(api.Spec), ContentLength: int64(len(api.Spec))}, nil
}

// loadLocation loads an IANA time zone, refusing the machine's own ("Local").
func loadLocation(name string) (*time.Location, error) {
	if name == "Local" {
		return nil, fmt.Errorf("unknown IANA time zone %q", name)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown IANA time zone %q", name)
	}
	return loc, nil
}

func cardOptions(p oapi.GetGreetingParams, lang greeting.Lang) card.Options {
	o := card.Defaults()
	o.Lang, o.Zone, o.Radius, o.Lat, o.Lon = lang, p.Tz, p.Radius, p.Lat, p.Lon
	if p.Layout != nil {
		o.Layout = card.Layout(*p.Layout)
	}
	if p.Show != nil {
		o.Facts = make([]card.Fact, 0, len(*p.Show))
		for _, f := range *p.Show {
			o.Facts = append(o.Facts, card.Fact(f))
		}
	}
	if p.Theme != nil {
		o.Theme = card.Theme(*p.Theme)
	}
	if p.Bg != nil {
		o.Colors.Bg = *p.Bg
	}
	if p.Accent != nil {
		o.Colors.Accent = *p.Accent
	}
	if p.Text != nil {
		o.Colors.Text = *p.Text
	}
	if p.Border != nil {
		o.Border = *p.Border
	}
	if p.Animate != nil {
		o.Animate = *p.Animate
	}
	if p.Clock != nil {
		o.Clock = card.Clock(*p.Clock)
	}
	if p.Width != nil {
		o.Width = *p.Width
	}
	return o
}

func greetingJSON(r greeting.Result, lang greeting.Lang, zone string) oapi.Greeting {
	sun := oapi.Sun{
		State:           oapi.SunState(r.Today.State),
		DaylightMinutes: int(r.Today.Daylight().Round(time.Minute) / time.Minute),
	}
	if r.Today.State == greeting.SunNormal {
		rise, set := r.Today.Sunrise, r.Today.Sunset
		sun.Sunrise, sun.Sunset = &rise, &set
	}
	if p, ok := r.Progress(); ok {
		p = round3(p)
		sun.Progress = &p
	}
	m := astro.MoonAt(r.Now)
	return oapi.Greeting{
		Moon:       oapi.Moon{Phase: oapi.MoonPhase(m.Phase()), Illumination: round3(m.Illumination), Waxing: m.Waxing},
		Greeting:   greeting.Text(lang, r.Period),
		Period:     oapi.Period(r.Period),
		Lang:       oapi.Lang(lang),
		Timezone:   zone,
		UtcOffset:  r.Now.Format("-07:00"),
		LocalTime:  r.Now.Truncate(time.Second),
		NextChange: r.NextChange,
		Sun:        sun,
	}
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// etagOf is a strong validator of a response body.
func etagOf(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

// matches reports whether an If-None-Match header names etag, with the weak
// comparison RFC 9110 prescribes for it.
func matches(ifNoneMatch *string, etag string) bool {
	if ifNoneMatch == nil {
		return false
	}
	for _, tag := range strings.Split(*ifNoneMatch, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "*" || strings.TrimPrefix(tag, "W/") == etag {
			return true
		}
	}
	return false
}

func badRequest(errs ...oapi.ParameterError) oapi.GetGreeting400ApplicationProblemPlusJSONResponse {
	return oapi.GetGreeting400ApplicationProblemPlusJSONResponse(newProblem(400, errs))
}
