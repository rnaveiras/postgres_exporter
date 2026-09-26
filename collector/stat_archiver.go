package collector

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query.
	statArchiver = `
SELECT archived_count
     , failed_count
     , stats_reset
  FROM pg_stat_archiver /*postgres_exporter*/`
)

type statArchiverScraper struct {
	archivedCount *prometheus.Desc
	failedCount   *prometheus.Desc
	statsReset    *prometheus.Desc
}

// NewStatarchiverScraper returns a new Scraper exposing PostgreSQL `pg_stat_archiver` view.
func NewStatArchiverScraper() Scraper {
	return &statArchiverScraper{
		archivedCount: prometheus.NewDesc(
			"postgres_stat_archiver_archived_total",
			"Number of WAL files that have been successfully archived",
			nil,
			nil,
		),
		failedCount: prometheus.NewDesc(
			"postgres_stat_archiver_failed_total",
			"Number of failed attempts for archiving WAL files",
			nil,
			nil,
		),
		statsReset: prometheus.NewDesc(
			"postgres_stat_archiver_stats_reset_timestamp",
			"Time at which these statistics were last reset",
			nil,
			nil,
		),
	}
}

func (*statArchiverScraper) Name() string {
	return "StatArchiverScraper"
}

func (c *statArchiverScraper) Scrape(ctx context.Context, db Querier, _ Version, ch chan<- prometheus.Metric) error {
	var archivedCount, failedCount int64
	// stats_reset is NULL until the statistics are reset for the first time.
	var statsReset pgtype.Timestamptz

	if err := db.QueryRow(ctx, statArchiver).
		Scan(&archivedCount,
			&failedCount,
			&statsReset,
		); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(c.archivedCount, prometheus.CounterValue, float64(archivedCount))
	ch <- prometheus.MustNewConstMetric(c.failedCount, prometheus.CounterValue, float64(failedCount))
	if statsReset.Valid {
		ch <- prometheus.MustNewConstMetric(c.statsReset, prometheus.GaugeValue, float64(statsReset.Time.Unix()))
	}
	return nil
}
