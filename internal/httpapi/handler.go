package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/ruanpato/polite-greetings/api"
	"github.com/ruanpato/polite-greetings/internal/oapi"
)

// Config is what NewHandler needs.
type Config struct {
	Now    func() time.Time // the clock
	Logger *slog.Logger
	Static fs.FS // the page, served at /
}

// NewHandler returns the whole HTTP handler: the API, checked against the
// contract, and the page, behind security headers and an access log.
func NewHandler(c Config) (http.Handler, error) {
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec)
	if err != nil {
		return nil, fmt.Errorf("load the contract: %w", err)
	}
	// The contract checks wrap the API routes from the outside, so they run
	// before the generated code binds the parameters and report every invalid
	// parameter at once. The page's files are not in the contract.
	validate := nethttpmiddleware.OapiRequestValidatorWithOptions(doc, &nethttpmiddleware.Options{
		Options:               openapi3filter.Options{MultiError: true},
		ErrorHandlerWithOpts:  validationProblem,
		SilenceServersWarning: true,
		DoNotValidateServers:  true,
		Skipper:               func(r *http.Request) bool { return !strings.HasPrefix(r.URL.Path, "/api/") },
	})
	logger := c.Logger
	strict := oapi.NewStrictHandlerWithOptions(&Server{Now: c.Now}, nil, oapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeProblem(w, newProblem(http.StatusBadRequest, nil, err.Error()))
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("response failed", "path", r.URL.Path, "error", err)
			writeProblem(w, newProblem(http.StatusInternalServerError, nil))
		},
	})
	mux := http.NewServeMux()
	oapi.HandlerWithOptions(strict, oapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: bindingProblem})
	mux.Handle("GET /", http.FileServerFS(c.Static))
	return securityHeaders(accessLog(logger, validate(mux))), nil
}

// bindingProblem reports a parameter the generated code could not bind. The
// contract checks catch these first; this keeps the answer the same shape if
// one ever slips through.
func bindingProblem(w http.ResponseWriter, _ *http.Request, err error) {
	var (
		required *oapi.RequiredParamError
		format   *oapi.InvalidParamFormatError
		header   *oapi.RequiredHeaderError
		many     *oapi.TooManyValuesForParamError
		unmarsh  *oapi.UnmarshalingParamError
	)
	var param oapi.ParameterError
	switch {
	case errors.As(err, &required):
		param = oapi.ParameterError{Parameter: required.ParamName, Detail: "is required"}
	case errors.As(err, &format):
		param = oapi.ParameterError{Parameter: format.ParamName, Detail: "has an invalid format"}
	case errors.As(err, &header):
		param = oapi.ParameterError{Parameter: header.ParamName, Detail: "is invalid"}
	case errors.As(err, &many):
		param = oapi.ParameterError{Parameter: many.ParamName, Detail: "is given more than once"}
	case errors.As(err, &unmarsh):
		param = oapi.ParameterError{Parameter: unmarsh.ParamName, Detail: "is invalid"}
	default:
		writeProblem(w, newProblem(http.StatusBadRequest, nil, err.Error()))
		return
	}
	writeProblem(w, newProblem(http.StatusBadRequest, []oapi.ParameterError{param}))
}

// securityHeaders sets headers that cost nothing. API responses, SVG cards
// included, may load nothing but their own inline styles; the page may load
// only its own files.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") || strings.HasSuffix(r.URL.Path, ".js") || strings.HasSuffix(r.URL.Path, ".css") {
			h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		} else {
			h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		}
		next.ServeHTTP(w, r)
	})
}

// accessLog logs one line per request. It leaves the query string out: it
// carries the coordinates of whoever uses the card.
func accessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status,
			"bytes", rec.bytes, "duration_ms", time.Since(start).Milliseconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status, bytes int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// validationProblem turns the contract checks' errors into problem details,
// one entry per invalid parameter.
func validationProblem(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
	if opts.StatusCode != http.StatusBadRequest {
		writeProblem(w, newProblem(opts.StatusCode, nil))
		return
	}
	var errs []error
	var multi openapi3.MultiError
	if errors.As(err, &multi) {
		errs = multi
	} else {
		errs = []error{err}
	}
	var params []oapi.ParameterError
	for _, e := range errs {
		var re *openapi3filter.RequestError
		if errors.As(e, &re) && re.Parameter != nil {
			params = append(params, oapi.ParameterError{Parameter: re.Parameter.Name, Detail: reason(re)})
			continue
		}
		params = append(params, oapi.ParameterError{Parameter: "", Detail: e.Error()})
	}
	writeProblem(w, newProblem(http.StatusBadRequest, params))
}

// reason is the short explanation of a parameter error, without the parameter
// name and location that the problem entry already gives.
func reason(re *openapi3filter.RequestError) string {
	msg := re.Reason
	var se *openapi3.SchemaError
	switch {
	case errors.As(re.Err, &se) && se.Reason != "":
		msg = se.Reason
	case re.Err != nil:
		msg = re.Err.Error()
	}
	// The JSON Schema 2020-12 validator, used for OpenAPI 3.1, writes a header
	// line and "- at '<pointer>': " before each finding.
	var findings []string
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(line, "jsonschema validation failed") {
			continue
		}
		findings = append(findings, schemaPointer.ReplaceAllString(strings.TrimSpace(line), ""))
	}
	return strings.Join(findings, "; ")
}

var schemaPointer = regexp.MustCompile(`^- at '[^']*': `)

func newProblem(status int, params []oapi.ParameterError, detail ...string) oapi.Problem {
	p := oapi.Problem{Type: "about:blank", Title: http.StatusText(status), Status: status}
	switch {
	case len(detail) > 0:
		d := strings.Join(detail, " ")
		p.Detail = &d
	case len(params) == 1:
		d := "1 parameter is invalid."
		p.Detail = &d
	case len(params) > 1:
		d := fmt.Sprintf("%d parameters are invalid.", len(params))
		p.Detail = &d
	}
	if len(params) > 0 {
		p.Errors = &params
	}
	return p
}

func writeProblem(w http.ResponseWriter, p oapi.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}
