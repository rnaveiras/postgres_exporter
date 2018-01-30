package collector

import (
	"context"
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	infoQuery         = `SHOW server_version /*postgres_exporter*/`
	upQuery           = `SELECT 1 /*postgres_exporter*/`
	isInRecoveryQuery = `SELECT pg_is_in_recovery() /*postgres_exporter*/`
)

type infoCollector struct {
	up           *prometheus.Desc
	info         *prometheus.Desc
	isInRecovery *prometheus.Desc
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

		isInRecovery: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "in_recovery"),
			"Postgres pg_is_in_recovery() True if recovery is still in progress.",
			nil,
			nil,
		),
	}, nil
}

func (c *infoCollector) Update(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric) error {
	var up, version, recovery string
	r := map[string]int{"t": 1, "f": 0}

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

	if err := db.QueryRowContext(ctx, isInRecoveryQuery).Scan(&recovery); err != nil {
		return err
	}

	// postgres_recovery
	ch <- prometheus.MustNewConstMetric(c.isInRecovery, prometheus.GaugeValue, float64(r[recovery]))

	return nil
}
