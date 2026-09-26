package collector

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
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
		 , pg_table_size(relid)::float AS size
  FROM pg_stat_user_tables /*postgres_exporter*/`
)

type diskUsageScraper struct {
	indexUsage *prometheus.Desc
	tableUsage *prometheus.Desc
}

// NewDiskUsageScraper returns a new Scraper exposing postgres disk usage view.
func NewDiskUsageScraper() Scraper {
	return &diskUsageScraper{
		indexUsage: prometheus.NewDesc(
			"postgres_disk_usage_index_bytes",
			"Bytes used on disk to store this index",
			[]string{labelDatname, labelSchemaname, "tablename", labelIndexname},
			nil,
		),
		tableUsage: prometheus.NewDesc(
			"postgres_disk_usage_table_bytes",
			"Bytes used on disk to store this table",
			[]string{labelDatname, labelSchemaname, "tablename"},
			nil,
		),
	}
}

func (*diskUsageScraper) Name() string {
	return "DiskUsageScraper"
}

func (c *diskUsageScraper) Scrape(ctx context.Context, conn Querier, _ Version, ch chan<- prometheus.Metric) error {
	var datname string
	if err := conn.QueryRow(ctx, currentDatabaseQuery).Scan(&datname); err != nil {
		return err
	}
	if err := c.scrapeTables(ctx, conn, datname, ch); err != nil {
		return err
	}
	return c.scrapeIndexes(ctx, conn, datname, ch)
}

func (c *diskUsageScraper) scrapeTables(ctx context.Context, conn Querier, datname string, ch chan<- prometheus.Metric) error {
	rows, err := conn.Query(ctx, tableUsageQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var schemaname, tablename string
	// sizeBytes is NULL for a relation dropped between the catalog read and the size call.
	var sizeBytes pgtype.Float8
	for rows.Next() {
		if err := rows.Scan(&schemaname, &tablename, &sizeBytes); err != nil {
			return err
		}
		if !sizeBytes.Valid {
			continue
		}

		// postgres_disk_usage_table_bytes
		ch <- prometheus.MustNewConstMetric(c.tableUsage, prometheus.GaugeValue, sizeBytes.Float64, datname, schemaname, tablename)
	}
	return rows.Err()
}

func (c *diskUsageScraper) scrapeIndexes(ctx context.Context, conn Querier, datname string, ch chan<- prometheus.Metric) error {
	rows, err := conn.Query(ctx, indexUsageQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var schemaname, tablename, indexname string
	// sizeBytes is NULL for an index dropped between the catalog read and the size call.
	var sizeBytes pgtype.Float8
	for rows.Next() {
		if err := rows.Scan(&schemaname, &tablename, &indexname, &sizeBytes); err != nil {
			return err
		}
		if !sizeBytes.Valid {
			continue
		}

		// postgres_disk_usage_index_bytes
		ch <- prometheus.MustNewConstMetric(c.indexUsage, prometheus.GaugeValue, sizeBytes.Float64, datname, schemaname, tablename, indexname)
	}
	return rows.Err()
}
