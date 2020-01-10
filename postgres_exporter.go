package main

import (
	stdlog "log"
	"net/http"
	"os"
	"sync"

	// _ "net/http/pprof"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	pgx "github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/rnaveiras/postgres_exporter/collector"
	"github.com/rnaveiras/postgres_exporter/gokitadapter"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	defaultDSN string = "user=postgres host=/var/run/postgresql"
)

var handlerLock sync.Mutex

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default("0.0.0.0:9187").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	dataSource    = kingpin.Flag("db.data-source", "libpq compatible data source").Default(defaultDSN).String()
	logLevel      = kingpin.Flag("log.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]").Default("info").String()
)

func init() {
	prometheus.MustRegister(version.NewCollector("postgres_exporter"))
}

func main() {
	kingpin.Version(version.Print("postgres_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	l, err := setlogLevel(*logLevel)
	if err != nil {
		level.Error(logger).Log("error", err)
		os.Exit(1)
	}

	logger = level.NewFilter(logger, l)
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)
	stdlog.SetOutput(kitlog.NewStdlibAdapter(logger))

	level.Info(logger).Log("msg", "Starting Postgres exporter", "version", version.Info())
	level.Info(logger).Log("build_context", version.BuildContext())

	connConfig, err := pgx.ParseConfig(*dataSource)
	if err != nil {
		level.Error(logger).Log("error", err)
		os.Exit(1)
	}

	level.Info(logger).Log("user", connConfig.User, "host", connConfig.Host, "dbname", connConfig.Database)

	connConfig.RuntimeParams = map[string]string{
		"client_encoding":  "UTF8",
		"application_name": "postgres_exporter",
	}

	connConfig.Logger = gokitadapter.NewLogger(logger)
	connConfig.LogLevel = pgx.LogLevelNone

	if *logLevel != "info" {
		connConfig.LogLevel, err = pgx.LogLevelFromString(*logLevel)
		if err != nil {
			level.Error(logger).Log("error", err)
			os.Exit(1)
		}
	}

	http.Handle(*metricsPath, metricsHandler(logger, connConfig))
	http.Handle("/", catchHandler(metricsPath))

	level.Info(logger).Log("component", "web", "msg", "Start listening for connections", "address", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		level.Error(logger).Log("error", err)
		os.Exit(1)
	}
}

func catchHandler(meticsPath *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html>
               <head><title>postgres Exporter</title></head>
               <body>
               <h1>Postgres Exporter</h1>
               <p><a href="` + *metricsPath + `">Metrics</a></p>
               </body>
               </html>`))
	})
}

func metricsHandler(logger kitlog.Logger, connConfig *pgx.ConnConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerLock.Lock()
		defer handlerLock.Unlock()

		registry := prometheus.NewRegistry()
		registry.MustRegister(collector.NewExporter(r.Context(), logger, connConfig))

		gatherers := prometheus.Gatherers{
			prometheus.DefaultGatherer,
			registry, // postgres_exporter metrics
		}

		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
}

func setlogLevel(s string) (level.Option, error) {
	var o level.Option
	switch s {
	case "debug":
		o = level.AllowDebug()
	case "info":
		o = level.AllowInfo()
	case "warn":
		o = level.AllowWarn()
	case "error":
		o = level.AllowError()
	default:
		return level.AllowAll(), errors.Errorf("unrecognized log level %q", s)
	}

	return o, nil
}
