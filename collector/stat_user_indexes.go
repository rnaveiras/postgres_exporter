package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

// The Statistics Scraper
// PostgreSQL's statistics collector is a subsystem that supports collection and reporting of information about
// server activity. Presently, the collector can count accesses to tables and indexes in both disk-block and
// individual-row terms. It also tracks the total number of rows in each table, and information about vacuum
// and analyze actions for each table. It can also count calls to user-defined functions and the total time
// spent in each one.
// https://www.postgresql.org/docs/9.4/static/monitoring-stats.html#PG-STAT-ALL-INDEXES-VIEW
const (
	// Scrape query
	statUserIndexesQuery = `
SELECT schemaname
     , relname
     , indexrelname
     , idx_scan::float
     , idx_tup_read::float
     , idx_tup_fetch::float
  FROM pg_stat_user_indexes
 WHERE schemaname != 'information_schema'
  AND idx_tup_fetch IS NOT NULL /*postgres_exporter*/`
)

type statUserIndexesScraper struct {
	idxScan     *prometheus.Desc
	idxTupRead  *prometheus.Desc
	idxTupFetch *prometheus.Desc
}

// NewStatUserIndexesScraper returns a new Scraper exposing postgres pg_stat_user_indexes view
func NewStatUserIndexesScraper() Scraper {
	return &statUserIndexesScraper{
		idxScan: prometheus.NewDesc(
			"postgres_stat_user_indexes_scan_total",
			"Number of times this index has been scanned",
			[]string{"datname", "schemaname", "relname", "indexname"},
			nil,
		),
		idxTupRead: prometheus.NewDesc(
			"postgres_stat_user_indexes_tuple_read_total",
			"Number of times tuples have been returned from scanning this index",
			[]string{"datname", "schemaname", "relname", "indexname"},
			nil,
		),
		idxTupFetch: prometheus.NewDesc(
			"postgres_stat_user_indexes_tuple_fetch_total",
			"Number of live tuples fetched by scans on this index",
			[]string{"datname", "schemaname", "relname", "indexname"},
			nil,
		),
	}
}

func (c *statUserIndexesScraper) Name() string {
	return "StatUserIndexesScraper"
}

func (c *statUserIndexesScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	var datname string
	if err := conn.QueryRow(ctx, "SELECT current_database() /*postgres_exporter*/").Scan(&datname); err != nil {
		return err
	}

	rows, err := conn.Query(ctx, statUserIndexesQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var schemaname, relname, indexname string
	var idxScan, idxTupRead, idxTupFetch float64
	for rows.Next() {
		if err := rows.Scan(&schemaname,
			&relname,
			&indexname,
			&idxScan,
			&idxTupRead,
			&idxTupFetch); err != nil {
			return err
		}

		// postgres_stat_user_indexes_idx_scan_total
		ch <- prometheus.MustNewConstMetric(c.idxScan, prometheus.CounterValue, idxScan, datname, schemaname, relname, indexname)
		// postgres_stat_user_indexes_idx_tup_read_total
		ch <- prometheus.MustNewConstMetric(c.idxTupRead, prometheus.CounterValue, idxTupRead, datname, schemaname, relname, indexname)
		// postgres_stat_user_indexes_idx_tup_fetch_total
		ch <- prometheus.MustNewConstMetric(c.idxTupFetch, prometheus.CounterValue, idxTupFetch, datname, schemaname, relname, indexname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
