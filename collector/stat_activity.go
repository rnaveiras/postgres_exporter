package collector

import (
	"context"
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Subsystem
	statActivitySubsystem = "stat_activity"

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
	   SELECT datname, state, count(*)
       FROM pg_stat_activity GROUP BY datname, state
	   ) AS activity
USING (datname, state) /*postgres_exporter*/`

	// Oldest transaction timestamp
	// ignore when backend_xid is null, so excludes autovacuumn, autoanalyze
	// and other maintenance tasks
	statActivityCollectorXactQuery = `
SELECT COALESCE(min(xact_start), current_timestamp) AS xact_start
  FROM pg_stat_activity
  WHERE state IN ('idle in transaction', 'active')
  AND backend_xid IS NOT NULL /*postgres_exporter*/`

	// Oldest backend timestamp
	statActivityCollectorBackendStartQuery = `SELECT min(backend_start) FROM pg_stat_activity /*postgres_exporter*/`

	// Oldest query in running state (long queries)"
	statActivityCollectorActiveQuery = `
SELECT EXTRACT(EPOCH FROM age(clock_timestamp(), min(query_start)))
  FROM pg_stat_activity /*postgres_exporter*/`
)

type statActivityCollector struct {
	connections *prometheus.Desc
	xact        *prometheus.Desc
	backend     *prometheus.Desc
	activeQuery *prometheus.Desc
}

func init() {
	registerCollector("stat_activity", defaultEnabled, NewStatActivityCollector)
}

// NewStatActivityCollector returns a new Collector exposing postgres pg_stat_activity
func NewStatActivityCollector() (Collector, error) {
	return &statActivityCollector{
		connections: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statActivitySubsystem, "connections"),
			"Number of current connections in their current state",
			[]string{"datname", "state"},
			nil,
		),
		xact: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statActivitySubsystem, "oldest_xact_timestamp"),
			"The oldest transaction started timestamp",
			nil,
			nil,
		),
		backend: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statActivitySubsystem, "oldest_backend_timestamp"),
			"The oldest backend started timestamp",
			nil,
			nil,
		),
		activeQuery: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statActivitySubsystem, "oldest_query_active_seconds"),
			"The oldest query in state running (long query)",
			nil,
			nil,
		),
	}, nil
}

func (c *statActivityCollector) Update(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.QueryContext(ctx, statActivityQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var datname, state string
	var count float64
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

	var oldestTx, oldestBackend time.Time
	err = db.QueryRowContext(ctx, statActivityCollectorXactQuery).Scan(&oldestTx)
	if err != nil {
		return err
	}

	// postgres_stat_activity_oldest_xact_timestamp
	ch <- prometheus.MustNewConstMetric(c.xact, prometheus.GaugeValue, float64(oldestTx.UTC().Unix()))

	err = db.QueryRowContext(ctx, statActivityCollectorBackendStartQuery).Scan(&oldestBackend)
	if err != nil {
		return err
	}

	// postgres_stat_activity_oldest_backend_timestamp
	ch <- prometheus.MustNewConstMetric(c.backend, prometheus.GaugeValue, float64(oldestBackend.UTC().Unix()))

	var duration float64
	err = db.QueryRowContext(ctx, statActivityCollectorActiveQuery).Scan(&duration)
	if err != nil {
		return err
	}

	// postgres_stat_activty_oldest_query_active_seconds
	ch <- prometheus.MustNewConstMetric(c.activeQuery, prometheus.GaugeValue, duration)

	return nil
}
