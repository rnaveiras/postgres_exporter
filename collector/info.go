package collector

import (
	"context"
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	infoQuery = `SHOW server_version`
	upQuery   = `SELECT 1`
)

type infoCollector struct {
	up   *prometheus.Desc
	info *prometheus.Desc
}

func init() {
	registerCollector("info", defaultEnabled, NewInfoCollector)
}

// NewInfoCollector returns a new Collector exposing postgres info
func NewInfoCollector() (Collector, error) {
	return &infoCollector{
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Whether the Postgres server is up.",
			nil,
			nil,
		),
		info: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "info"),
			"Postgres version and distribution.",
			[]string{"version"},
			nil,
		),
	}, nil
}

func (c *infoCollector) Update(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric) error {
	var up, version string

	err := db.Ping()
	if err != nil {
		// postgres_up
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return err
	}

	err = db.QueryRowContext(ctx, upQuery).Scan(&up)
	if err != nil {
		// postgres_up
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return err
	}

	// postgres_up
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

	if err := db.QueryRowContext(ctx, infoQuery).Scan(&version); err != nil {
		return err
	}

	// postgres_info
	ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, version)

	return nil
}
