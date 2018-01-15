package collector

import (
	"context"
	"database/sql"

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
						USING (datname, state);
						`
)

type statActivityCollector struct {
	connections *prometheus.Desc
}

func init() {
	registerCollector("stat_activity", defaultEnabled, NewStatActivityCollector)
}

// NewStatActivityColletor returns a new Collector expsoing postgres pg_stat_activity
func NewStatActivityCollector() (Collector, error) {
	return &statActivityCollector{
		connections: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statActivitySubsystem, "connections"),
			"Number of current connections in their current state",
			[]string{"datname", "state"},
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

	return nil
}
