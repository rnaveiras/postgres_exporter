package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	versionBitSize   = 64
	infoQuery        = `SHOW server_version /*postgres_exporter*/`
	listDatnameQuery = `
SELECT datname FROM pg_database
WHERE datallowconn = true AND datistemplate = false
AND datname != ALL($1) /*postgres_exporter*/`
	successValue    = 1.0
	failureValue    = 0.0
	infoMetricValue = 1.0
	errorKey        = "error"
)

var (
	upDesc = prometheus.NewDesc(
		"postgres_up",
		"Whether the Postgres server is up.",
		nil,
		nil,
	)
	infoDesc = prometheus.NewDesc(
		"postgres_info",
		"Postgres version",
		[]string{"version"},
		nil,
	)
	scrapeDurationDesc = prometheus.NewDesc(
		"postgres_exporter_scraper_duration_seconds",
		"Duration of a scrapers scrape.",
		[]string{"scraper", "datname"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		"postgres_exporter_scraper_success",
		"Whether a scraper succeeded.",
		[]string{"scraper", "datname"},
		nil,
	)
)

// Scraper is the interface each scraper has to implement.
type Scraper interface {
	Name() string
	// Scrape new metrics and expose them via prometheus registry.
	Scrape(ctx context.Context, db *pgx.Conn, version Version, ch chan<- prometheus.Metric) error
}

type Exporter struct {
	ctx               context.Context
	logger            *slog.Logger
	connConfig        *pgx.ConnConfig
	scrapers          []Scraper
	datnameScrapers   []Scraper
	excludedDatabases []string
}

// Postgres Version
type Version struct {
	version float64
}

func NewVersion(v string) Version {
	values := strings.Split(v, " ")
	version, _ := strconv.ParseFloat(values[0], versionBitSize)
	return Version{
		version: version,
	}
}

func (v Version) Gte(n float64) bool {
	return v.version >= n
}

func (v Version) String() string {
	return fmt.Sprintf("%g", v.version)
}

// Verify our Exporter satisfies the prometheus.Collector interface
var _ prometheus.Collector = (*Exporter)(nil)

// NewExporter is called every time we receive a scrape request and knows how
// to collect metrics using each of the scrapers. It will live only for the
// duration of the scrape request.
func NewExporter(ctx context.Context, logger *slog.Logger, connConfig *pgx.ConnConfig, excludedDatabases []string) *Exporter {
	return &Exporter{
		ctx:        ctx,
		logger:     logger,
		connConfig: connConfig,
		scrapers: []Scraper{
			NewInfoScraper(),
			NewLocksScraper(),
			NewStatActivityScraper(),
			NewStatArchiverScraper(),
			NewStatBgwriterScraper(),
			NewStatDatabaseScraper(),
			NewStatReplicationScraper(),
		},
		datnameScrapers: []Scraper{
			NewStatVacuumProgressScraper(),
			NewStatUserTablesScraper(),
			NewStatUserIndexesScraper(),
			NewDiskUsageScraper(),
		},
		excludedDatabases: excludedDatabases,
	}
}

// Describe implements the prometheus.Collector interface.
func (Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	conn, err := pgx.ConnectConfig(e.ctx, e.connConfig)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, failureValue)
		e.logger.Error("exporter collect",
			slog.Any(errorKey, err))
		return // cannot continue without a valid connection
	}

	defer conn.Close(e.ctx)
	// postgres_up
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, successValue)

	var version string
	if err := conn.QueryRow(e.ctx, infoQuery).Scan(&version); err != nil {
		e.logger.Error("info query",
			slog.Any(errorKey, err))
		return // cannot continue without a version
	}

	v := NewVersion(version)
	// postgres_info
	ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, infoMetricValue, v.String())

	// discovery databases
	e.logger.Debug("excluded databases",
		slog.String("databases", strings.Join(e.excludedDatabases, ",")))

	rows, err := conn.Query(e.ctx, listDatnameQuery, e.excludedDatabases)
	if err != nil {
		e.logger.Error("error query datnames",
			slog.Any(errorKey, err))
	}

	var dbnames []string
	dbnames, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		e.logger.Error("error list datname query",
			slog.Any(errorKey, err))
		return
	}

	e.logger.Debug("debug datnames found",
		slog.String("databases", strings.Join(dbnames, ",")))

	// run global scrapers
	for _, scraper := range e.scrapers {
		e.scrape(scraper, conn, v, ch)
	}

	// run datname scrapers
	for _, dbname := range dbnames {
		// update connection dbname
		e.connConfig.Database = dbname

		// establish a new connection
		conn, err := pgx.ConnectConfig(e.ctx, e.connConfig)
		if err != nil {
			e.logger.Error("error pgx connection",
				slog.Any(errorKey, err))
			return // cannot continue without a valid connection
		}

		// scrape
		for _, scraper := range e.datnameScrapers {
			e.scrape(scraper, conn, v, ch)
		}

		conn.Close(e.ctx)
	}
}

func (e *Exporter) scrape(scraper Scraper, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) {
	start := time.Now()
	err := scraper.Scrape(e.ctx, conn, version, ch)
	duration := time.Since(start)

	var success float64

	logger := e.logger.With(
		"scraper", scraper.Name(),
		"duration", duration.Seconds())
	if err != nil {
		logger.Error("failed scrape",
			slog.Any(errorKey, err))
		success = failureValue
	} else {
		logger.Debug("",
			"event", "scraper.success")
		success = successValue
	}

	datname := e.connConfig.Database
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), scraper.Name(), datname)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, scraper.Name(), datname)
}
