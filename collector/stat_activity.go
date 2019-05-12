package collector

import (
	"context"
	"time"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	statActivityQuery = `
WITH states AS (
  SELECT datname
	   , unnest(array['active',
					  'idle',
					  'idle in transaction',
					  'idle in transaction (aborted)',
					  'fastpath function call',
					  'disabled']) AS state FROM pg_database
)
SELECT datname, state, COALESCE(count, 0) as count
  FROM states LEFT JOIN (
	   SELECT datname, state, count(*)::float
       FROM pg_stat_activity GROUP BY datname, state
	   ) AS activity
 USING (datname, state) /*postgres_exporter*/`

	// Oldest transaction timestamp
	// ignore when backend_xid is null, so excludes autovacuumn, autoanalyze
	// and other maintenance tasks
	statActivityScraperXactQuery = `
SELECT EXTRACT(EPOCH FROM age(clock_timestamp(), coalesce(min(xact_start), current_timestamp))) AS xact_start
  FROM pg_stat_activity
 WHERE state IN ('idle in transaction', 'active')
   AND backend_xid IS NOT NULL /*postgres_exporter*/`

	// Oldest backend timestamp
	statActivityScraperBackendStartQuery = `SELECT min(backend_start) FROM pg_stat_activity /*postgres_exporter*/`

	// Oldest query in running state (long queries)"
	statActivityScraperActiveQuery = `
SELECT EXTRACT(EPOCH FROM age(clock_timestamp(), coalesce(min(query_start), clock_timestamp())))
  FROM pg_stat_activity
 WHERE state='active' /*postgres_exporter*/`

	// Oldest Snapshot
	// ignore when backend_xid is null, so we exclude autovacuumn, autoanalyze
	// and other maintenance tasks
	statActivityScraperOldestSnapshotQuery = `
SELECT EXTRACT(EPOCH FROM age(clock_timestamp(), coalesce(min(query_start), clock_timestamp())))
  FROM pg_stat_activity
 WHERE backend_xmin IS NOT NULL
   AND backend_xid  IS NOT NULL /*postgres_exporter*/`
)

type statActivityScraper struct {
	connections *prometheus.Desc
	backend     *prometheus.Desc
	xact        *prometheus.Desc
	active      *prometheus.Desc
	snapshot    *prometheus.Desc
}

// NewStatActivityScraper returns a new Scraper exposing postgres pg_stat_activity
func NewStatActivityScraper() Scraper {
	return &statActivityScraper{
		connections: prometheus.NewDesc(
			"postgres_stat_activity_connections",
			"Number of current connections in their current state",
			[]string{"datname", "state"},
			nil,
		),
		backend: prometheus.NewDesc(
			"postgres_stat_activity_oldest_backend_timestamp",
			"The oldest backend started timestamp",
			nil,
			nil,
		),
		xact: prometheus.NewDesc(
			"postgres_stat_activity_oldest_xact_seconds",
			"The oldest transaction (active or idle in transaction)",
			nil,
			nil,
		),
		active: prometheus.NewDesc(
			"postgres_stat_activity_oldest_query_active_seconds",
			"The oldest query in running state (long query)",
			nil,
			nil,
		),
		snapshot: prometheus.NewDesc(
			"postgres_stat_activity_oldest_snapshot_seconds",
			"The oldest snapshot",
			nil,
			nil,
		),
	}
}

func (c *statActivityScraper) Name() string {
	return "StatActivityScraper"
}

func (c *statActivityScraper) Scrape(ctx context.Context, conn *pgx.Conn, ch chan<- prometheus.Metric) error {
	rows, err := conn.QueryEx(ctx, statActivityQuery, nil)
	if err != nil {
		return err
	}
	defer rows.Close()

	var datname, state string
	var count, oldestTx, oldestActive, oldestSnapshot float64
	var oldestBackend time.Time

	for rows.Next() {
		if err := rows.Scan(&datname, &state, &count); err != nil {
			return err
		}

		// postgres_stat_activity_connections
		ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, count, datname, state)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	err = conn.QueryRowEx(ctx, statActivityScraperBackendStartQuery, nil).Scan(&oldestBackend)
	if err != nil {
		return err
	}

	// postgres_stat_activity_oldest_backend_timestamp
	ch <- prometheus.MustNewConstMetric(c.backend, prometheus.GaugeValue, float64(oldestBackend.UTC().Unix()))

	err = conn.QueryRowEx(ctx, statActivityScraperXactQuery, nil).Scan(&oldestTx)
	if err != nil {
		return err
	}

	// postgres_stat_activity_oldest_xact_seconds
	ch <- prometheus.MustNewConstMetric(c.xact, prometheus.GaugeValue, oldestTx)

	err = conn.QueryRowEx(ctx, statActivityScraperActiveQuery, nil).Scan(&oldestActive)
	if err != nil {
		return err
	}

	// postgres_stat_activity_oldest_query_active_seconds
	ch <- prometheus.MustNewConstMetric(c.active, prometheus.GaugeValue, oldestActive)

	err = conn.QueryRowEx(ctx, statActivityScraperOldestSnapshotQuery, nil).Scan(&oldestSnapshot)
	if err != nil {
		return err
	}

	// postgres_stat_activity_oldest_snapshot_seconds
	ch <- prometheus.MustNewConstMetric(c.snapshot, prometheus.GaugeValue, oldestSnapshot)

	return nil
}
