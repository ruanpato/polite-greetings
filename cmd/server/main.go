// Command server runs polite-greetings: the API in api/openapi.yaml, its docs
// and the page.
//
// It listens on the port in PORT (8080 by default), logs JSON lines to stdout
// and shuts down gracefully on SIGINT or SIGTERM: /readyz turns 503 first, then
// the requests in flight finish.
//
// `server healthcheck` asks a running server whether it is ready and exits 0 or
// 1, for container health checks on images without curl.
package main

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the time zone database, for images without one

	"github.com/ruanpato/polite-greetings/internal/buildinfo"
	"github.com/ruanpato/polite-greetings/internal/httpapi"
	"github.com/ruanpato/polite-greetings/web"
)

func main() {
	port := cmp.Or(os.Getenv("PORT"), "8080")
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck("http://127.0.0.1:" + port + "/readyz"))
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, port); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, port string) error {
	readiness := &httpapi.Readiness{}
	commit := buildinfo.Commit()
	handler, err := httpapi.NewHandler(httpapi.Config{
		Now: time.Now, Logger: logger, Static: web.FS, Commit: commit, Readiness: readiness,
	})
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errs := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr, "commit", commit)
		errs <- srv.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
	}
	readiness.ShutDown()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
