package collector

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	infoQuery         = `SHOW server_version /*postgres_exporter*/`
	upQuery           = `SELECT 1 /*postgres_exporter*/`
	isInRecoveryQuery = `SELECT pg_is_in_recovery()::int /*postgres_exporter*/`
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

func (c *infoCollector) Update(ctx context.Context, db *pgx.Conn, ch chan<- prometheus.Metric) error {
	var version string
	var recovery int64

	err := db.Ping(ctx)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return err
	}

	rows, _ := db.QueryEx(ctx, upQuery, nil)
	rows.Close()
	if rows.Err() != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return err
	}

	if db.IsAlive() {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)
	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
	}

	if err := db.QueryRowEx(ctx, infoQuery, nil).Scan(&version); err != nil {
		return err
	}
	// postgres_info
	ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, version)

	if err := db.QueryRowEx(ctx, isInRecoveryQuery, nil).Scan(&recovery); err != nil {
		return err
	}
	// postgres_recovery
	ch <- prometheus.MustNewConstMetric(c.isInRecovery, prometheus.GaugeValue, float64(recovery))

	return nil
}
