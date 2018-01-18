package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	// _ "net/http/pprof"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/rnaveiras/postgres_exporter/collector"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	db            *sql.DB
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9187").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	dataSource    = kingpin.Flag("db.data-source", "libpq compatible data source").Envar("DATA_SOURCE_NAME").Default("postgresql:///postgres?host=/var/run/postgresql").String()
)

func init() {
	prometheus.MustRegister(version.NewCollector("postgres_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debugln("collect query:", filters)

	c, err := collector.NewPostgresCollector(r.Context(), db, filters...)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(c)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}

	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError,
	})
	h.ServeHTTP(w, r)
}

func main() {
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("postgres_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting postgres_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	dsn, err := pq.ParseURL(*dataSource)
	if err != nil {
		log.Fatal("parse dsn:", err)
	}
	log.Debugln("connection string: ", dsn)

	// Open Postgres connection
	db, err = sql.Open("postgres", *dataSource)
	log.Infoln("Established a new database connection.")

	if err != nil {
		log.Errorln("Error openning connection to database:", err)
	}

	// By design exporter should use maximum one connection per scrape
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	// Reuse the same connection forever
	db.SetConnMaxLifetime(0)
	defer db.Close()

	ctx := context.Background()
	// This instance is only used to check collector creation and logging.
	c, err := collector.NewPostgresCollector(ctx, db)
	if err != nil {
		log.Fatalf("Couldn't create collector: %s", err)
	}
	log.Infof("Enabled collectors:")
	for n := range c.Collectors {
		log.Infof(" - %s", n)
	}

	// TODO: Remove deprecated InstrumentHandlerFunc usage.
	http.HandleFunc(*metricsPath, prometheus.InstrumentHandlerFunc("metrics", handler))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>postgres Exporter</title></head>
			<body>
			<h1>Postgres Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
