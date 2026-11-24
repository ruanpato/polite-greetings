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

	"github.com/getkin/kin-openapi/openapi3"

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
	return handlerWith(t, Config{Logger: slog.New(slog.NewJSONHandler(logs, nil))})
}

// handlerWith builds the handler on a fixed clock and stand-ins for the page's
// files, with the rest of the configuration from c.
func handlerWith(t *testing.T, c Config) http.Handler {
	t.Helper()
	c.Now = func() time.Time { return fixedNow }
	if c.Logger == nil {
		c.Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	c.Static = fstest.MapFS{
		"index.html":  {Data: []byte("<!doctype html><title>page</title>")},
		"docs.html":   {Data: []byte("<!doctype html><title>docs</title>")},
		"favicon.svg": {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
	}
	h, err := NewHandler(c)
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

// TestHead checks that HEAD answers like GET, without the body, over a real
// connection (the recorder used elsewhere does not drop HEAD bodies).
func TestHead(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(t, nil))
	defer srv.Close()
	url := srv.URL + "/api/v2/greeting?" + brasilia
	getResp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	getResp.Body.Close()
	etag := getResp.Header.Get("ETag")

	for _, ifNoneMatch := range []string{"", etag} {
		req, _ := http.NewRequest(http.MethodHead, url, nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		want := http.StatusOK
		if ifNoneMatch != "" {
			want = http.StatusNotModified
		}
		if resp.StatusCode != want || len(body) != 0 || resp.Header.Get("ETag") != etag {
			t.Errorf("HEAD (If-None-Match %q): status %d, %d body bytes, ETag %q; want %d, no body, %q",
				ifNoneMatch, resp.StatusCode, len(body), resp.Header.Get("ETag"), want, etag)
		}
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
	// 24 November 2026 is a full moon.
	if g.Moon.Phase != oapi.FullMoon || g.Moon.Illumination < 0.97 {
		t.Errorf("moon = %+v, want a full moon", g.Moon)
	}
}

func TestCosmosCard(t *testing.T) {
	h := newTestHandler(t, nil)
	rec := get(t, h, "/api/v2/greeting?"+brasilia+"&layout=cosmos&show=sun,moon&lang=pt-BR")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	for _, want := range []string{"Boa tarde", "Lua cheia · 100%", `clip-path="url(#pg-globe)"`, `class="n"`} {
		if !strings.Contains(body, want) {
			t.Errorf("cosmos card misses %q", want)
		}
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
		{brasilia + "&show=sun,stars", []string{"show"}},
		{brasilia + "&layout=globe", []string{"layout"}},
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

func TestMetaRoutesAndPages(t *testing.T) {
	h := newTestHandler(t, nil)
	spec := get(t, h, "/openapi.yaml")
	if spec.Code != http.StatusOK || !bytes.Equal(spec.Body.Bytes(), api.Spec) || spec.Header().Get("Content-Type") != "application/yaml" {
		t.Errorf("openapi.yaml: %d, %s, %d bytes", spec.Code, spec.Header().Get("Content-Type"), spec.Body.Len())
	}
	page := get(t, h, "/")
	pageCSP := page.Header().Get("Content-Security-Policy")
	if page.Code != http.StatusOK || !strings.Contains(pageCSP, "default-src 'self'") || strings.Contains(pageCSP, "jsdelivr") {
		t.Errorf("page: %d, CSP %q", page.Code, pageCSP)
	}
	// The docs page is the only one allowed to load scripts from jsDelivr.
	docs := get(t, h, "/docs")
	docsCSP := docs.Header().Get("Content-Security-Policy")
	if docs.Code != http.StatusOK || !strings.Contains(docs.Body.String(), "<title>docs</title>") ||
		!strings.Contains(docsCSP, "script-src 'self' https://cdn.jsdelivr.net;") {
		t.Errorf("docs: %d, CSP %q, body %s", docs.Code, docsCSP, docs.Body)
	}
	if rec := get(t, h, "/docs/"); rec.Code != http.StatusNotFound {
		t.Errorf("/docs/: %d", rec.Code)
	}
	icon := get(t, h, "/favicon.svg")
	if icon.Code != http.StatusOK || icon.Header().Get("Content-Type") != "image/svg+xml" {
		t.Errorf("favicon: %d, %s", icon.Code, icon.Header().Get("Content-Type"))
	}
	if rec := get(t, h, "/api/v2/nothing"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown path: %d", rec.Code)
	}
}

func TestHealthChecks(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec)
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	readiness := &Readiness{}
	h := handlerWith(t, Config{
		Logger:    slog.New(slog.NewJSONHandler(&logs, nil)),
		Commit:    "3e86c94d6f4e1d0a8f5b2c7e9a1b3d5f7c9e0a2b",
		Readiness: readiness,
	})
	check := func(path string, wantCode int, want oapi.Health) {
		t.Helper()
		rec := get(t, h, path)
		var got oapi.Health
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: %v in %s", path, err, rec.Body)
		}
		if rec.Code != wantCode || got != want {
			t.Errorf("%s: %d %+v, want %d %+v", path, rec.Code, got, wantCode, want)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("%s: Cache-Control %q", path, cc)
		}
	}

	ok := oapi.Health{Status: oapi.Ok, Version: doc.Info.Version, Commit: "3e86c94d6f4e1d0a8f5b2c7e9a1b3d5f7c9e0a2b"}
	check("/healthz", http.StatusOK, ok)
	check("/readyz", http.StatusOK, ok)

	// Shutting down: no new traffic, but the process is still alive.
	readiness.ShutDown()
	shuttingDown := ok
	shuttingDown.Status = oapi.ShuttingDown
	check("/readyz", http.StatusServiceUnavailable, shuttingDown)
	check("/healthz", http.StatusOK, ok)

	if logs.Len() > 0 {
		t.Errorf("health checks were logged: %s", logs.String())
	}

	// Without a known commit, nor a readiness switch.
	rec := get(t, handlerWith(t, Config{}), "/readyz")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"commit":"unknown"`) {
		t.Errorf("no commit: %d %s", rec.Code, rec.Body)
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
