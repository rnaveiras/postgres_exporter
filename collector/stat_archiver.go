package collector

import (
	"context"
	"time"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	statArchiver = `
SELECT archived_count
     , failed_count
     , stats_reset
  FROM pg_stat_archiver /*postgres_exporter*/`
)

type statArchiverCollector struct {
	archivedCount *prometheus.Desc
	failedCount   *prometheus.Desc
	statsReset    *prometheus.Desc
}

func init() {
	registerCollector("stat_archiver", defaultEnabled, NewStatArchiverCollector)
}

// NewStatarchiverCollector returns a new Collector exposing PostgreSQL `pg_stat_archiver` view
func NewStatArchiverCollector() (Collector, error) {
	return &statArchiverCollector{
		archivedCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "stat_archiver_archived_total"),
			"Number of WAL files that have been successfully archived",
			nil,
			nil,
		),
		failedCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "stat_archiver_failed_total"),
			"Number of failed attempts for archiving WAL files",
			nil,
			nil,
		),
		statsReset: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "stat_archiver_stats_reset_timestamp"),
			"Time at which these statistics were last reset",
			nil,
			nil,
		),
	}, nil
}

func (c *statArchiverCollector) Update(ctx context.Context, db *pgx.Conn, ch chan<- prometheus.Metric) error {
	var archivedCount, failedCount int64
	var statsReset time.Time

	if err := db.QueryRowEx(ctx, statArchiver, nil).
		Scan(&archivedCount,
			&failedCount,
			&statsReset,
		); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(c.archivedCount, prometheus.CounterValue, float64(archivedCount))
	ch <- prometheus.MustNewConstMetric(c.failedCount, prometheus.CounterValue, float64(failedCount))
	ch <- prometheus.MustNewConstMetric(c.statsReset, prometheus.GaugeValue, float64(statsReset.UTC().Unix()))
	return nil
}
