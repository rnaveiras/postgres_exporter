package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	indexUsageQuery = `
	SELECT schemaname
		, relname AS tablename
		, indexrelname AS indexname
		, pg_relation_size(indexrelid)::float AS size
  FROM pg_stat_user_indexes /*postgres_exporter*/`

	tableUsageQuery = `
	SELECT schemaname
		 , relname AS tablename
		 , pg_table_size(schemaname || '.' || relname)::float AS size
  FROM pg_stat_user_tables /*postgres_exporter*/`
)

type diskUsageScraper struct {
	indexUsage *prometheus.Desc
	tableUsage *prometheus.Desc
}

// NewDiskUsageScraper returns a new Scraper exposing postgres disk usage view
func NewDiskUsageScraper() Scraper {
	return &diskUsageScraper{
		indexUsage: prometheus.NewDesc(
			"postgres_disk_usage_index_bytes",
			"Bytes used on disk to store this index",
			[]string{"datname", "schemaname", "tablename", "indexname"},
			nil,
		),
		tableUsage: prometheus.NewDesc(
			"postgres_disk_usage_table_bytes",
			"Bytes used on disk to store this table",
			[]string{"datname", "schemaname", "tablename"},
			nil,
		),
	}
}

func (*diskUsageScraper) Name() string {
	return "DiskUsageScraper"
}

func (c *diskUsageScraper) Scrape(ctx context.Context, conn *pgx.Conn, _ Version, ch chan<- prometheus.Metric) error {
	var datname string
	var rows pgx.Rows
	var err error

	if err := conn.QueryRow(ctx, "SELECT current_database() /*postgres_exporter*/").Scan(&datname); err != nil {
		return err
	}

	// Scan table sizes
	rows, err = conn.Query(ctx, tableUsageQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	type tableRow struct {
		schemaname string
		tablename  string
		sizeBytes  float64
	}

	tableRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[tableRow])
	if err != nil {
		return err
	}

	// postgres_disk_usage_table_bytes
	for _, row := range tableRows {
		ch <- prometheus.MustNewConstMetric(c.tableUsage, prometheus.GaugeValue, row.sizeBytes, datname, row.schemaname, row.tablename)
	}

	// Scan index bytes
	rows, err = conn.Query(ctx, indexUsageQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	type indexRow struct {
		schemaname string
		tablename  string
		indexname  string
		sizeBytes  float64
	}

	indexRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[indexRow])
	if err != nil {
		return err
	}

	// postgres_disk_usage_index_bytes
	for _, row := range indexRows {
		ch <- prometheus.MustNewConstMetric(c.indexUsage, prometheus.GaugeValue, row.sizeBytes, datname, row.schemaname, row.tablename, row.indexname)
	}

	return nil
}
