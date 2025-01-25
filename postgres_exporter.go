package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	// _ "net/http/pprof"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/rnaveiras/postgres_exporter/collector"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	readHeaderTimeout = 3 * time.Second
	levelError        = "error" // Used for error logging
	exitCodeError     = 1
)

var handlerLock sync.Mutex

var (
	listenAddress = kingpin.Flag("web.listen-address",
		"Address on which to expose metrics and web interface.").Default("0.0.0.0:9187").String()
	metricsPath = kingpin.Flag("web.telemetry-path",
		"Path under which to expose metrics.").Default("/metrics").String()
	dataSource = kingpin.Flag("db.data-source",
		"libpq compatible data source, e.g `user=postgres host=/var/run/postgresql`. Leave blank for libpq envs").String()
	logLevel = kingpin.Flag("log.level",
		"Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]").Default("info").String()
)

func main() {
	kingpin.Version(version.Print("postgres_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logLevel := new(slog.LevelVar)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: false,
	}))
	slog.SetDefault(logger)
	// l, err := setlogLevel(*logLevel)
	// if err != nil {
	// level.Error(logger).Log(levelError, err)
	// os.Exit(exitCodeError)
	// }

	logger.Info("Starting Postgres exporter", "version", version.Info())
	logger.Info("", "build_context", version.BuildContext())

	connConfig, err := pgx.ParseConfig(*dataSource)
	if err != nil {
		logger.Error("failed parse connection string", slog.Any("error", err))
		os.Exit(exitCodeError)
	}
	connConfig.Tracer = &tracelog.TraceLog{
		Logger:   &SlogAdapter{logger: logger},
		LogLevel: tracelog.LogLevelNone,
	}

	logger.Info("connection string", "user", connConfig.User, "host", connConfig.Host, "dbname", connConfig.Database)

	connConfig.RuntimeParams = map[string]string{
		"client_encoding":  "UTF8",
		"application_name": "postgres_exporter",
	}

	// connConfig.Logger = gokitadapter.NewLogger(logger)
	// connConfig.LogLevel = pgx.LogLevelNone

	// if *logLevel != "info" {
	// connConfig.LogLevel, err = pgx.LogLevelFromString(*logLevel)
	// if err != nil {
	// level.Error(logger).Log(levelError, err)
	// os.Exit(exitCodeError)
	// }
	// }

	http.Handle(*metricsPath, metricsHandler(logger, connConfig))
	http.Handle("/", catchHandler(logger, metricsPath))
	http.Handle("/log/level", logLevelHandler(logger, logLevel))

	logger.Info("Start listening for connections", "component", "web", "address", *listenAddress)

	server := &http.Server{
		Addr:              *listenAddress,
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(_ net.Listener) context.Context { return context.Background() },
	}
	err = server.ListenAndServe()
	if err != nil {
		logger.Error("failed listen and server", slog.Any("error", err))
	}
}

func catchHandler(logger *slog.Logger, metricsPath *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`<html>
               <head><title>postgres Exporter</title></head>
               <body>
               <h1>Postgres Exporter</h1>
               <p><a href="` + *metricsPath + `">Metrics</a></p>
               </body>
               </html>`))
		logger.Error("catch all handler", slog.Any("error", err))
	})
}

func metricsHandler(logger *slog.Logger, connConfig *pgx.ConnConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerLock.Lock()
		defer handlerLock.Unlock()

		registry := prometheus.NewRegistry()
		registry.MustRegister(versioncollector.NewCollector("postgres_exporter"))
		registry.MustRegister(collector.NewExporter(r.Context(), logger, connConfig))

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

//	func setlogLevel(s string) (level.Option, error) {
//		var o level.Option
//		switch s {
//		case "debug":
//			o = level.AllowDebug()
//		case "info":
//			o = level.AllowInfo()
//		case "warn":
//			o = level.AllowWarn()
//		case "error":
//			o = level.AllowError()
//		default:
//			return level.AllowAll(), errors.Errorf("unrecognized log level %q", s)
//		}
//
//		return o, nil
//	}
func logLevelHandler(logger *slog.Logger, logLevel *slog.LevelVar) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type logLevelRequest struct {
			Level string `json:"level"`
		}

		if r.Method != http.MethodPatch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req logLevelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var level slog.Level
		switch req.Level {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			http.Error(w, "Invalid log level", http.StatusBadRequest)
			return
		}

		logLevel.Set(level)
		logger.Info("change logger level", "level", level)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logLevelRequest{Level: req.Level})
	})
}
