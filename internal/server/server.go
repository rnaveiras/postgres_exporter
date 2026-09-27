package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
	"sync/atomic"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rnaveiras/postgres_exporter/collector"
)

const (
	errorKey = "error"

	// Concurrent /metrics requests served before promhttp returns 503.
	maxRequestsInFlight = 15

	// Server timeouts.
	readTimeout       = 5 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 120 * time.Second
	readHeaderTimeout = 5 * time.Second

	// Graceful shutdown timeout.
	shutdownTimeout = 30 * time.Second
)

var handlerLock sync.Mutex

// Config is what the exporter serves and how it reaches Postgres. The command line in
// cmd/postgres_exporter builds it.
type Config struct {
	// ListenAddress is the address the HTTP server binds, e.g. "0.0.0.0:9187" or "127.0.0.1:0".
	ListenAddress string
	// MetricsPath is the path of the metrics endpoint.
	MetricsPath string
	// DataSource is a libpq connection string; empty uses the PG* environment variables.
	DataSource string
	// ExcludedDatabases are never scraped by the per-database scrapers.
	ExcludedDatabases []string
	// LegacyNames also emits deprecated metric names next to their replacements.
	LegacyNames bool
	// Pprof serves the Go runtime profiling endpoints under /debug/pprof/.
	Pprof bool
	// AdminAPI serves /-/log-level, which changes the log level at runtime.
	AdminAPI bool
}

// Run serves the exporter until ctx is canceled. A canceled ctx is a clean shutdown and returns nil; failing to
// start or to keep serving returns the error. logLevel is the level logger filters on; /-/log-level changes it.
func Run(ctx context.Context, cfg Config, logger *slog.Logger, logLevel *slog.LevelVar) error {
	s, err := newServer(ctx, cfg, logger, logLevel)
	if err != nil {
		return err
	}
	return s.serve(ctx)
}

// server is the exporter's HTTP server, bound to its listener but not yet serving.
type server struct {
	logger   *slog.Logger
	listener net.Listener
	http     *http.Server
	ready    *atomic.Bool
}

// newServer builds the connection config and binds the listen address. Binding before serving makes a busy
// address fail the start.
func newServer(ctx context.Context, cfg Config, logger *slog.Logger, logLevel *slog.LevelVar) (*server, error) {
	connConfig, err := newConnConfig(cfg.DataSource, logger)
	if err != nil {
		return nil, err
	}

	logger = logger.With("component", "web")
	if cfg.AdminAPI {
		logger.Warn("admin API enabled, its endpoints have no authentication", "path", logLevelPath)
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", cfg.ListenAddress, err)
	}
	logger.Info("start listening for connections",
		"address", listener.Addr().String(),
	)

	ready := new(atomic.Bool)
	return &server{
		logger:   logger,
		listener: listener,
		ready:    ready,
		http: &http.Server{
			Handler:           newMux(logger, connConfig, cfg, logLevel, ready),
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ReadHeaderTimeout: readHeaderTimeout,

			// Scrapes outlive no request; a canceled ctx stops them through Shutdown, not through this.
			BaseContext: func(_ net.Listener) context.Context { return context.WithoutCancel(ctx) },
		},
	}, nil
}

func (s *server) addr() string {
	return s.listener.Addr().String()
}

// serve serves until ctx is canceled, then shuts down gracefully. It returns an error when the server stops
// on its own, so the process exits instead of staying up without serving.
func (s *server) serve(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() { errc <- s.http.Serve(s.listener) }()

	s.ready.Store(true)
	s.logger.Info("ready")

	select {
	case err := <-errc:
		s.ready.Store(false)
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	s.logger.Info("shutting down server - received signal",
		errorKey, ctx.Err())
	s.ready.Store(false)

	// ctx is already canceled; shutdown gets its own deadline for in-flight requests.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	s.logger.Info("server gracefully stopped")
	return nil
}

// newConnConfig parses the data source and sets the tracer and session parameters every scrape uses.
func newConnConfig(dataSource string, logger *slog.Logger) (*pgx.ConnConfig, error) {
	connConfig, err := pgx.ParseConfig(dataSource)
	if err != nil {
		return nil, fmt.Errorf("parse data source: %w", err)
	}

	logger.Info("connection string",
		"user", connConfig.User,
		"host", connConfig.Host,
		"dbname", connConfig.Database,
		"port", connConfig.Port,
	)

	// Configure the connection tracer for PostgreSQL query logging
	// - Uses a custom SlogAdapter to integrate with our structured logging
	// - LogLevel is set to None by default to avoid excessive logging
	// This tracer can be used to debug database operations if needed
	// by changing the LogLevel to tracelog.LogLevelDebug
	connConfig.Tracer = &tracelog.TraceLog{
		Logger:   &SlogAdapter{logger: logger},
		LogLevel: tracelog.LogLevelNone,
	}

	// Set PostgreSQL session parameters for this connection:
	// - client_encoding: ensures proper character encoding (UTF8)
	// - application_name: identifies this connection in pg_stat_activity
	//   making it easier to track exporter connections in the database
	connConfig.RuntimeParams = map[string]string{
		"client_encoding":  "UTF8",
		"application_name": "postgres_exporter",
	}
	return connConfig, nil
}

// newMux registers the HTTP endpoints. The profiling endpoints exist only with --web.enable-pprof and the
// admin endpoints only with --web.enable-admin-api.
func newMux(logger *slog.Logger, connConfig *pgx.ConnConfig, cfg Config, logLevel *slog.LevelVar,
	ready *atomic.Bool,
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(cfg.MetricsPath, metricsHandler(logger, connConfig, cfg))
	mux.Handle("GET "+healthyPath, healthyHandler(logger))
	mux.Handle("GET "+readyPath, readyHandler(logger, ready))
	mux.Handle("/", catchHandler(logger, cfg.MetricsPath))

	if cfg.AdminAPI {
		mux.Handle(logLevelPath, newLogLevelControl(logger, logLevel))
	}

	if cfg.Pprof {
		// pprof.Index serves every named profile under /debug/pprof/; the others need their own handlers.
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	return mux
}

// catchHandler creates an HTTP handler that serves the index page of the exporter.
func catchHandler(logger *slog.Logger, metricsPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`<html>
               <head><title>postgres Exporter</title></head>
               <body>
               <h1>Postgres Exporter</h1>
               <p><a href="` + metricsPath + `">Metrics</a></p>
               </body>
               </html>`))
		if err != nil {
			logger.Error("catch all handler",
				slog.Any(errorKey, err))
		}
	})
}

// metricsHandler creates an HTTP handler that serves Prometheus metrics for PostgreSQL.
func metricsHandler(logger *slog.Logger, connConfig *pgx.ConnConfig, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerLock.Lock()
		defer handlerLock.Unlock()

		registry := prometheus.NewRegistry()
		registry.MustRegister(versioncollector.NewCollector("postgres_exporter"))
		registry.MustRegister(collector.NewExporter(r.Context(), logger, connConfig, collector.Options{
			ExcludedDatabases: cfg.ExcludedDatabases,
			LegacyNames:       cfg.LegacyNames,
		}))

		gatherers := prometheus.Gatherers{
			prometheus.DefaultGatherer,
			registry, // postgres_exporter metrics
		}

		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer,
			promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{
				ErrorHandling:       promhttp.ContinueOnError,
				Registry:            registry,
				MaxRequestsInFlight: maxRequestsInFlight,
			}))
		h.ServeHTTP(w, r)
	})
}
