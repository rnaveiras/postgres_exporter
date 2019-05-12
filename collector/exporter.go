package collector

import (
	"context"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	upDesc = prometheus.NewDesc(
		"postgres_up",
		"Whether the Postgres server is up.",
		nil,
		nil,
	)
	scrapeDurationDesc = prometheus.NewDesc(
		"postgres_exporter_scraper_duration_seconds",
		"Duration of a scrapers scrape.",
		[]string{"scraper"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		"postgres_exporter_scraper_success",
		"Whether a scraper succeeded.",
		[]string{"scraper"},
		nil,
	)
)

// Scraper is the interface each scraper has to implement.
type Scraper interface {
	Name() string
	// Scrape new metrics and expose them via prometheus registry.
	Scrape(ctx context.Context, db *pgx.Conn, ch chan<- prometheus.Metric) error
}

type Exporter struct {
	ctx        context.Context
	logger     kitlog.Logger
	connConfig pgx.ConnConfig
	scrapers   []Scraper
}

// Verify our Exporter satisfies the prometheus.Collector interface
var _ prometheus.Collector = (*Exporter)(nil)

// NewExporter is called every time we receive a scrape request and knows how
// to collect metrics using each of the scrapers. It will live only for the
// duration of the scrape request.
func NewExporter(ctx context.Context, logger kitlog.Logger, connConfig pgx.ConnConfig) *Exporter {
	return &Exporter{
		ctx:        ctx,
		logger:     logger,
		connConfig: connConfig,
		scrapers: []Scraper{
			NewInfoScraper(),
			NewLocksScraper(),
			NewStatActivityScraper(),
			NewStatArchiverScraper(),
			NewStatDatabaseScraper(),
			NewStatReplicationScraper(),
			NewStatUserTablesScraper(),
		},
	}
}

// Describe implements the prometheus.Collector interface.
func (e Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (e Exporter) Collect(ch chan<- prometheus.Metric) {
	conn, err := pgx.Connect(e.connConfig)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0)
		return // cannot continue without a valid connection
	}

	defer conn.Close()
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1)

	for _, scraper := range e.scrapers {
		e.scrape(scraper, conn, ch)
	}
}

func (e Exporter) scrape(scraper Scraper, conn *pgx.Conn, ch chan<- prometheus.Metric) {
	start := time.Now()
	err := scraper.Scrape(e.ctx, conn, ch)
	duration := time.Since(start)

	var success float64

	logger := kitlog.With(e.logger, "scraper", scraper.Name(), "duration", duration.Seconds())
	if err != nil {
		logger.Log("error", err)
		success = 0
	} else {
		level.Debug(logger).Log("event", "scraper.success")
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), scraper.Name())
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, scraper.Name())
}
