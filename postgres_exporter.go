package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/rnaveiras/postgres_exporter/collector"
)

const (
	readHeaderTimeout = 3 * time.Second
	errorKey          = "error"
	exitCodeError     = 1
)

var handlerLock sync.Mutex

type flagConfig struct {
	ListenAddress     string   `json:"listen_address"`
	MetricsPath       string   `json:"metrics_path"`
	DataSource        string   `json:"data_source"`
	LogLevel          string   `json:"log_level"`
	LogFormat         string   `json:"log_format"`
	Pprof             bool     `json:"pprof"`
	ExcludedDatabases []string `json:"excluded_databases"`
}

// LogValue implemnts LogValuer interface
func (f flagConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("listen_address", f.ListenAddress),
		slog.String("metrics_path", f.MetricsPath),
		slog.String("log_level", f.LogLevel),
		slog.String("log_format", f.LogFormat),
		slog.Bool("pprof", f.Pprof),
		slog.Any("exclude_databases", f.ExcludedDatabases),
	)
}

func main() {
	cfg := flagConfig{}

	a := kingpin.New(filepath.Base(os.Args[0]), "The Postgres Exporter").UsageWriter(os.Stdout)
	a.Version(version.Print("postgres_exporter"))
	a.HelpFlag.Short('h')

	a.Flag("web.listen-address", "Address on which to expose metrics and web interface.").
		Default("0.0.0.0:9187").StringVar(&cfg.ListenAddress)

	a.Flag("web.telemetry-path", "Path under which to expose metrics").
		Default("/metrics").StringVar(&cfg.MetricsPath)

	a.Flag("db.data-source", "libpq compatible connection string, e.g `user=postgres host=/var/run/postgresql`. Leave blank for libqp envs").
		StringVar(&cfg.DataSource)

	a.Flag("db.excluded-databases", "Repeat this flag for each database to exclude from monitoring").
		Default("cloudsdqladmin", "rdsadmin").StringsVar(&cfg.ExcludedDatabases)

	a.Flag("log.level", "Only log messages with the given severity or above. One of: [debug, info, warn, error]").
		Default("info").EnumVar(&cfg.LogLevel, "debug", "info", "warn", "error")

	a.Flag("log.format", "Output format of log messages. One of: [logfmt, json]").
		Default("logfmt").EnumVar(&cfg.LogFormat, "logfmt", "json")

	a.Flag("web.enabled-pprof", "").
		Default("false").BoolVar(&cfg.Pprof)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		//nolint:revive // Exiting anyway, so we can ignore
		fmt.Fprintln(os.Stderr, fmt.Errorf("error parsing command line arguments: %w", err))
		a.Usage(os.Args[1:])
		os.Exit(exitCodeError)
	}

	// Setup log level
	logLevel := new(slog.LevelVar)
	logger, err := setupLogger(logLevel, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		//nolint:revive // Exiting anyway, so we can ignore
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeError)
	}

	// Booting
	logger.Info("Starting Postgres exporter", "version", version.Info())
	logger.Info("", "build_context", version.BuildContext())

	// Log cfg configuration
	logger.Debug("cfg", "cfg", cfg)

	// ParseConfig creates a ConnConfig from a connection string.
	connConfig, err := pgx.ParseConfig(cfg.DataSource)
	if err != nil {
		logger.Error("error parse config", slog.Any(errorKey, err))
		os.Exit(exitCodeError)
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

	// Register HTTP endpoints
	http.Handle(cfg.MetricsPath, metricsHandler(logger, connConfig, cfg))
	http.Handle("/admin/loglevel", logLevelHandler(logger, logLevel))
	http.Handle("/", catchHandler(logger, cfg.MetricsPath))

	// Enable runtime profiling endpoints when pprof flag is set
	if cfg.Pprof {
		mux := http.NewServeMux()

		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		http.Handle("/debug/pprof", mux)
	}

	logger.Info("Start listening for connections",
		"component", "web",
		"address", cfg.ListenAddress,
	)

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(_ net.Listener) context.Context { return context.Background() },
	}
	err = server.ListenAndServe()
	if err != nil {
		logger.Error("failed listen and server", slog.Any(errorKey, err))
	}
}

// catchHandler creates an HTTP handler that serves the index page of the exporter.
func catchHandler(logger *slog.Logger, metricsPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`<html>
               <head><title>postgres Exporter</title></head>
               <body>
               <h1>Postgres Exporter</h1>
               <p><a href="` + metricsPath + `">Metrics</a></p>
               </body>
               </html>`))
		if err != nil {
			logger.Error("catch all handler", slog.Any(errorKey, err))
		}
	})
}

// metricsHandler creates an HTTP handler that serves Prometheus metrics for PostgreSQL.
func metricsHandler(logger *slog.Logger, connConfig *pgx.ConnConfig, cfg flagConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerLock.Lock()
		defer handlerLock.Unlock()

		registry := prometheus.NewRegistry()
		registry.MustRegister(versioncollector.NewCollector("postgres_exporter"))
		registry.MustRegister(collector.NewExporter(r.Context(), logger, connConfig, cfg.ExcludedDatabases))

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
				MaxRequestsInFlight: 15,
			}))
		h.ServeHTTP(w, r)
	})
}

// logLevelHandler creates an HTTP handler that enables dynamic log level
// adjustment
func logLevelHandler(logger *slog.Logger, logLevel *slog.LevelVar) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type logLevelJSON struct {
			Level string `json:"level"`
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			// Return current logLevel
			currentLevel := logLevel.Level().String()
			if err := json.NewEncoder(w).Encode(logLevelJSON{Level: currentLevel}); err != nil {
				http.Error(w, "error failed to encode JSON respose", http.StatusInternalServerError)
				return
			}

		case http.MethodPatch:
			var req logLevelJSON
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "error invalid request body", http.StatusBadRequest)
				return
			}

			// Validate user input
			validLevels := map[string]slog.Level{
				"debug": slog.LevelDebug,
				"info":  slog.LevelInfo,
				"warn":  slog.LevelWarn,
				"error": slog.LevelError,
			}
			level, ok := validLevels[strings.ToLower(req.Level)]
			if !ok {
				http.Error(w, "error invalid log level", http.StatusBadRequest)
				return
			}

			logLevel.Set(level)
			logger.Info("log level changed", "level", level)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// setupLogger configures the logger,
func setupLogger(logLevelVar *slog.LevelVar, logFormat string, logLevel string) (*slog.Logger, error) {
	// setup LogLevel
	if err := setLogLevel(logLevelVar, logLevel); err != nil {
		return nil, fmt.Errorf("error setting log level %w", err)
	}

	handlerOpts := slog.HandlerOptions{
		Level:     logLevelVar,
		AddSource: false,
	}

	var handler slog.Handler
	if logFormat == "logfmt" {
		handler = slog.NewTextHandler(os.Stderr, &handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, &handlerOpts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

// setLogLevel configures the log level from a string value.
// Valid levels are: debug, info, warn, error
func setLogLevel(logLevel *slog.LevelVar, level string) error {
	switch strings.ToLower(level) {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		return fmt.Errorf("invalid log level %q, valid levels are: debug, info, warn, error", level)
	}
	return nil
}
