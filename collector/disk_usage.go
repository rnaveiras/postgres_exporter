package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

// Queries taken from https://wiki.postgresql.org/wiki/Disk_Usage
const (
	relationUsageQuery = `
	SELECT
			nspname
		, relname
		, pg_relation_size(C.oid) AS "size_bytes"
  FROM pg_class C
  LEFT JOIN pg_namespace N ON (N.oid = C.relnamespace)
  WHERE nspname NOT IN ('pg_catalog', 'information_schema')
  ORDER BY pg_relation_size(C.oid) DESC /*postgres_exporter*/`

	tableUsageQuery = `
SELECT nspname
     , relname
     , pg_total_relation_size(C.oid) AS "total_size_bytes"
FROM pg_class C
LEFT JOIN pg_namespace N ON (N.oid = C.relnamespace)
WHERE nspname NOT IN ('pg_catalog', 'information_schema')
  AND C.relkind <> 'i'
  AND nspname !~ '^pg_toast'
ORDER BY pg_total_relation_size(C.oid) DESC /*postgres_exporter*/`
)

type diskUsageScraper struct {
	relationUsage *prometheus.Desc
	tableUsage    *prometheus.Desc
}

// NewDiskUsageScraper returns a new Scraper exposing postgres disk usage view
func NewDiskUsageScraper() Scraper {
	return &diskUsageScraper{
		relationUsage: prometheus.NewDesc(
			"postgres_disk_usage_relation_bytes",
			"Bytes used on disk to store this relation",
			[]string{"datname", "schemaname", "relname"},
			nil,
		),
		tableUsage: prometheus.NewDesc(
			"postgres_disk_usage_table_bytes",
			"Bytes used on disk to store this table (including indexes)",
			[]string{"datname", "schemaname", "relname"},
			nil,
		),
	}
}

func (c *diskUsageScraper) Name() string {
	return "DiskUsageScraper"
}

func (c *diskUsageScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	var datname, schemaname, relname string
	var sizeBytes uint64
	if err := conn.QueryRow(ctx, "SELECT current_database() /*postgres_exporter*/").Scan(&datname); err != nil {
		return err
	}

	rows, err := conn.Query(ctx, relationUsageQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&schemaname, &relname, &sizeBytes); err != nil {
			return err
		}

		// postgres_disk_usage_relation_bytes
		ch <- prometheus.MustNewConstMetric(c.relationUsage, prometheus.GaugeValue, float64(sizeBytes), datname, schemaname, relname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	rows, err = conn.Query(ctx, tableUsageQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&schemaname, &relname, &sizeBytes); err != nil {
			return err
		}

		// postgres_disk_usage_table_bytes
		ch <- prometheus.MustNewConstMetric(c.tableUsage, prometheus.GaugeValue, float64(sizeBytes), datname, schemaname, relname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
