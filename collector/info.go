package collector

import (
	"context"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	isInRecoveryQuery   = `SELECT pg_is_in_recovery()::int /*postgres_exporter*/`
	isInBackupQuery     = `SELECT pg_is_in_backup()::int /*postgres_exporter*/`
	startTimeQuery      = `SELECT pg_postmaster_start_time() /*postgres_exporter*/`
	configLoadTimeQuery = `SELECT pg_conf_load_time() /*postgres_exporter*/`
)

type infoScraper struct {
	isInRecovery   *prometheus.Desc
	isInBackup     *prometheus.Desc
	startTime      *prometheus.Desc
	configLoadTime *prometheus.Desc
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
		isInBackup: prometheus.NewDesc(
			"postgres_is_in_backup",
			"True if an on-line exclusive backup is still in progress.",
			nil,
			nil,
		),
		startTime: prometheus.NewDesc(
			"postgres_start_time_seconds",
			"Postgres start time, in seconds since the unix epoch.",
			nil,
			nil,
		),
		configLoadTime: prometheus.NewDesc(
			"postgres_config_last_load_time_seconds",
			"Timestamp of the last configuration reload",
			nil,
			nil,
		),
	}
}

func (*infoScraper) Name() string {
	return "InfoScraper"
}

func (c *infoScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	var recovery, backup int64
	var startTime, configLoadTime time.Time

	if err := conn.QueryRow(ctx, isInRecoveryQuery).Scan(&recovery); err != nil {
		return err
	}
	// postgres_is_in_recovery
	ch <- prometheus.MustNewConstMetric(c.isInRecovery, prometheus.GaugeValue, float64(recovery))

	if err := conn.QueryRow(ctx, isInBackupQuery).Scan(&backup); err != nil {
		return err
	}

	// postgres_is_in_backup was removed in PostgreSQL 15
	if !version.Gte(15.0) {
		var backup int64
		if err := conn.QueryRow(ctx, isInBackupQuery).Scan(&backup); err != nil {
			return err
		}
		// postgres_is_in_backup
		ch <- prometheus.MustNewConstMetric(c.isInBackup, prometheus.GaugeValue, float64(backup))
	}

	if err := conn.QueryRow(ctx, startTimeQuery).Scan(&startTime); err != nil {
		return err
	}
	// postgres_start_time_seconds
	ch <- prometheus.MustNewConstMetric(c.startTime, prometheus.GaugeValue, float64(startTime.UTC().Unix()))

	if err := conn.QueryRow(ctx, configLoadTimeQuery).Scan(&configLoadTime); err != nil {
		return err
	}
	// postgres_config_last_load_time_seconds
	ch <- prometheus.MustNewConstMetric(c.configLoadTime, prometheus.GaugeValue, float64(configLoadTime.UTC().Unix()))

	return nil
}
