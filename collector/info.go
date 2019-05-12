package collector

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	infoQuery         = `SHOW server_version /*postgres_exporter*/`
	isInRecoveryQuery = `SELECT pg_is_in_recovery()::int /*postgres_exporter*/`
)

type infoScraper struct {
	info         *prometheus.Desc
	isInRecovery *prometheus.Desc
}

// NewInfoScraper returns a new Scraper exposing postgres info
func NewInfoScraper() Scraper {
	return &infoScraper{
		info: prometheus.NewDesc(
			"postgres_info",
			"Postgres version and distribution.",
			[]string{"version"},
			nil,
		),
		isInRecovery: prometheus.NewDesc(
			"postgres_is_in_recovery",
			"Postgres pg_is_in_recovery() True if recovery is still in progress.",
			nil,
			nil,
		),
	}
}

func (c *infoScraper) Name() string {
	return "InfoScraper"
}

func (c *infoScraper) Scrape(ctx context.Context, conn *pgx.Conn, ch chan<- prometheus.Metric) error {
	var version string
	var recovery int64

	if err := conn.QueryRowEx(ctx, infoQuery, nil).Scan(&version); err != nil {
		return err
	}
	// postgres_info
	ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, version)

	if err := conn.QueryRowEx(ctx, isInRecoveryQuery, nil).Scan(&recovery); err != nil {
		return err
	}
	// postgres_recovery
	ch <- prometheus.MustNewConstMetric(c.isInRecovery, prometheus.GaugeValue, float64(recovery))

	return nil
}
