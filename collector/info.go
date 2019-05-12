package collector

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	isInRecoveryQuery = `SELECT pg_is_in_recovery()::int /*postgres_exporter*/`
)

type infoScraper struct {
	isInRecovery *prometheus.Desc
}

// NewInfoScraper returns a new Scraper exposing postgres info
func NewInfoScraper() Scraper {
	return &infoScraper{
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

func (c *infoScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	var recovery int64

	if err := conn.QueryRowEx(ctx, isInRecoveryQuery, nil).Scan(&recovery); err != nil {
		return err
	}
	// postgres_recovery
	ch <- prometheus.MustNewConstMetric(c.isInRecovery, prometheus.GaugeValue, float64(recovery))

	return nil
}
