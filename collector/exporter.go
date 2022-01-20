package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
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

const infoQuery = `SHOW server_version /*postgres_exporter*/`
const listDatnameQuery = `
SELECT datname FROM pg_database WHERE datallowconn = true AND datistemplate = false /*postgres_exporter*/`

// Scraper is the interface each scraper has to implement.
type Scraper interface {
	Name() string
	// Scrape new metrics and expose them via prometheus registry.
	Scrape(ctx context.Context, db *pgx.Conn, version Version, ch chan<- prometheus.Metric) error
}

type Exporter struct {
	logger          kitlog.Logger
	connConfig      *pgx.ConnConfig
	scrapers        []Scraper
	datnameScrapers []Scraper
}

type ExporterRun struct {
	ctx      context.Context
	exporter *Exporter
}

// Postgres Version
type Version struct {
	version float64
}

func NewVersion(v string) Version {
	values := strings.Split(v, " ")
	version, _ := strconv.ParseFloat(values[0], 64)
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

// Verify our ExporterRun satisfies the prometheus.Collector interface
var _ prometheus.Collector = ((*Exporter)(nil)).NewRun((context.Context)(nil))

// NewExporter is called every time we receive a scrape request and knows how
// to collect metrics using each of the scrapers. It will live only for the
// duration of the scrape request.
func NewExporter(logger kitlog.Logger, connConfig *pgx.ConnConfig) *Exporter {
	return &Exporter{
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
			NewCachedScraper(NewDiskUsageScraper(), 5*time.Minute),
		},
	}
}

func (e *Exporter) NewRun(ctx context.Context) *ExporterRun {
	return &ExporterRun{ctx: ctx, exporter: e}
}

// Describe implements the prometheus.Collector interface.
func (er ExporterRun) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (er ExporterRun) Collect(ch chan<- prometheus.Metric) {
	conn, err := pgx.ConnectConfig(er.ctx, er.exporter.connConfig)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0)
		level.Error(er.exporter.logger).Log("error", err)
		return // cannot continue without a valid connection
	}

	defer conn.Close(er.ctx)
	// postgres_up
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1)

	var version string
	if err := conn.QueryRow(er.ctx, infoQuery).Scan(&version); err != nil {
		level.Error(er.exporter.logger).Log("error", err)
		return // cannot continue without a version
	}

	v := NewVersion(version)
	// postgres_info
	ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, 1, v.String())

	// discovery databases
	var dbnames []string

	rows, err := conn.Query(er.ctx, listDatnameQuery)
	if err != nil {
		level.Error(er.exporter.logger).Log("error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var dbname string
		err := rows.Scan(&dbname)
		if err != nil {
			level.Error(er.exporter.logger).Log("error", err)
			return
		}
		dbnames = append(dbnames, dbname)
	}

	// run global scrapers
	for _, scraper := range er.exporter.scrapers {
		scraper.Scrape(er.ctx, conn, v, ch)
	}

	// run datname scrapers
	for _, dbname := range dbnames {
		// update connection dbname
		er.exporter.connConfig.Config.Database = dbname

		// establish a new connection
		conn, err := pgx.ConnectConfig(er.ctx, er.exporter.connConfig)
		if err != nil {
			level.Error(er.exporter.logger).Log("error", err)
			return // cannot continue without a valid connection
		}

		// scrape
		for _, scraper := range er.exporter.datnameScrapers {
			scraper.Scrape(er.ctx, conn, v, ch)
		}

		conn.Close(er.ctx)
	}
}

func (er ExporterRun) scrape(scraper Scraper, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) {
	start := time.Now()
	err := scraper.Scrape(er.ctx, conn, version, ch)
	duration := time.Since(start)

	var success float64

	logger := kitlog.With(er.exporter.logger, "scraper", scraper.Name(), "duration", duration.Seconds())
	if err != nil {
		logger.Log("error", err)
		success = 0
	} else {
		level.Debug(logger).Log("event", "scraper.success")
		success = 1
	}

	datname := er.exporter.connConfig.Config.Database
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), scraper.Name(), datname)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, scraper.Name(), datname)
}
