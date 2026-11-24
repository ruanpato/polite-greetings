package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ruanpato/polite-greetings/api"
	"github.com/ruanpato/polite-greetings/internal/oapi"
)

// 15:00 in Brasília on 24 November 2026: afternoon, sunset at about 18:27.
var fixedNow = time.Date(2026, 11, 24, 18, 0, 0, 0, time.UTC)

const brasilia = "tz=America/Sao_Paulo&lat=-15.78&lon=-47.93"

func newTestHandler(t *testing.T, logs io.Writer) http.Handler {
	t.Helper()
	if logs == nil {
		logs = io.Discard
	}
	h, err := NewHandler(Config{
		Now:    func() time.Time { return fixedNow },
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
		Static: fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>page</title>")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func get(t *testing.T, h http.Handler, target string, header ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGreetingCard(t *testing.T) {
	h := newTestHandler(t, nil)
	rec := get(t, h, "/api/v2/greeting?"+brasilia+"&layout=sky&show=sun,daylength,time")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	for name, want := range map[string]string{
		"Content-Type":            "image/svg+xml; charset=utf-8",
		"Cache-Control":           "no-cache",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if etag := rec.Header().Get("ETag"); !strings.HasPrefix(etag, `"`) || len(etag) != 18 {
		t.Errorf("ETag = %q, want a quoted strong tag", etag)
	}
	body := rec.Body.String()
	for _, want := range []string{"Good afternoon", ">15:00<", ">05:31<", ">18:27<", "12h 56m of daylight"} {
		if !strings.Contains(body, want) {
			t.Errorf("card misses %q", want)
		}
	}
}

func TestConditionalRequests(t *testing.T) {
	h := newTestHandler(t, nil)
	first := get(t, h, "/api/v2/greeting?"+brasilia)
	etag := first.Header().Get("ETag")
	for _, ifNoneMatch := range []string{etag, "W/" + etag, `"other", ` + etag, "*"} {
		rec := get(t, h, "/api/v2/greeting?"+brasilia, "If-None-Match", ifNoneMatch)
		if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
			t.Errorf("If-None-Match %s: status %d with %d bytes, want 304 and no body", ifNoneMatch, rec.Code, rec.Body.Len())
		}
		if rec.Header().Get("ETag") != etag || rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("If-None-Match %s: 304 without ETag or Cache-Control", ifNoneMatch)
		}
	}
	if rec := get(t, h, "/api/v2/greeting?"+brasilia, "If-None-Match", `"stale"`); rec.Code != http.StatusOK {
		t.Errorf("stale If-None-Match: status %d, want 200", rec.Code)
	}
	if other := get(t, h, "/api/v2/greeting?"+brasilia+"&theme=dark"); other.Header().Get("ETag") == etag {
		t.Error("a different card kept the same ETag")
	}
}

func TestTextAndJSON(t *testing.T) {
	h := newTestHandler(t, nil)
	txt := get(t, h, "/api/v2/greeting?"+brasilia+"&format=txt&lang=es")
	if txt.Body.String() != "Buenas tardes" || txt.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("txt = %q (%s)", txt.Body, txt.Header().Get("Content-Type"))
	}

	rec := get(t, h, "/api/v2/greeting?"+brasilia+"&format=json")
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
	var g oapi.Greeting
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	if g.Period != oapi.Afternoon || g.Greeting != "Good afternoon" || g.UtcOffset != "-03:00" || g.Timezone != "America/Sao_Paulo" {
		t.Errorf("greeting = %+v", g)
	}
	if g.Sun.State != oapi.Normal || g.Sun.Sunrise == nil || g.Sun.Sunset == nil || g.Sun.Progress == nil {
		t.Fatalf("sun = %+v", g.Sun)
	}
	if p := *g.Sun.Progress; p < 0.7 || p > 0.76 {
		t.Errorf("progress = %v, want about 0.733", p)
	}
	if !g.NextChange.Equal(*g.Sun.Sunset) {
		t.Errorf("next change %s, want the sunset %s", g.NextChange, *g.Sun.Sunset)
	}
}

func TestInvalidParameters(t *testing.T) {
	h := newTestHandler(t, nil)
	tests := []struct {
		query string
		want  []string // parameters named in the problem
	}{
		{"lat=-15.8&lon=-47.9", []string{"tz"}},
		{"tz=America/Sao_Paulo&lat=100&lon=-47.9", []string{"lat"}},
		{"tz=America/Sao_Paulo&lat=abc&lon=-47.9", []string{"lat"}},
		{"tz=America/Sao_Paulo&lat=100&lon=500", []string{"lat", "lon"}},
		{brasilia + "&layout=poster", []string{"layout"}},
		{brasilia + "&show=sun,moon", []string{"show"}},
		{brasilia + "&show=", []string{"show"}},
		{brasilia + "&bg=%23FFFFFF", []string{"bg"}},
		{brasilia + "&width=50", []string{"width"}},
		{brasilia + "&radius=99", []string{"radius"}},
		{"tz=Mars/Olympus_Mons&lat=0&lon=0", []string{"tz"}},
		{"tz=Local&lat=0&lon=0", []string{"tz"}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			rec := get(t, h, "/api/v2/greeting?"+tt.query)
			if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != "application/problem+json" {
				t.Fatalf("status %d (%s), body %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body)
			}
			var p oapi.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if p.Status != 400 || p.Title != "Bad Request" || p.Errors == nil {
				t.Fatalf("problem = %s", rec.Body)
			}
			var got []string
			for _, e := range *p.Errors {
				got = append(got, e.Parameter)
				if e.Detail == "" {
					t.Errorf("no detail for %s", e.Parameter)
				}
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("parameters = %v, want %v (%s)", got, tt.want, rec.Body)
			}
		})
	}
}

func TestMetaRoutesAndPage(t *testing.T) {
	h := newTestHandler(t, nil)
	if rec := get(t, h, "/healthz"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Errorf("healthz: %d %s", rec.Code, rec.Body)
	}
	spec := get(t, h, "/openapi.yaml")
	if spec.Code != http.StatusOK || !bytes.Equal(spec.Body.Bytes(), api.Spec) || spec.Header().Get("Content-Type") != "application/yaml" {
		t.Errorf("openapi.yaml: %d, %s, %d bytes", spec.Code, spec.Header().Get("Content-Type"), spec.Body.Len())
	}
	page := get(t, h, "/")
	if page.Code != http.StatusOK || !strings.Contains(page.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Errorf("page: %d, CSP %q", page.Code, page.Header().Get("Content-Security-Policy"))
	}
	if rec := get(t, h, "/api/v2/nothing"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown path: %d", rec.Code)
	}
}

func TestAccessLogLeavesCoordinatesOut(t *testing.T) {
	var logs bytes.Buffer
	h := newTestHandler(t, &logs)
	get(t, h, "/api/v2/greeting?"+brasilia)
	line := logs.String()
	if !strings.Contains(line, `"path":"/api/v2/greeting"`) || !strings.Contains(line, `"status":200`) {
		t.Errorf("log line = %s", line)
	}
	if strings.Contains(line, "-15.78") || strings.Contains(line, "Sao_Paulo") {
		t.Errorf("the log line carries the query string: %s", line)
	}
}
