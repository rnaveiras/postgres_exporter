package collector

import (
	"context"
	"strconv"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	locksSubsystem = "locks"
	locksQuery     = `
SELECT datname
     , locktype
     , mode
	 , granted
     , count(*)::float
  FROM pg_locks
  JOIN pg_database ON pg_locks.database=pg_database.oid
 GROUP BY datname, locktype, mode, granted /*postgres_exporter*/
`
)

type locksCollector struct {
	locks *prometheus.Desc
}

func init() {
	registerCollector("locks", defaultEnabled, NewLocksCollector)
}

// NewLocksCollector returns a new Collector exposing data from pg_locks
func NewLocksCollector() (Collector, error) {
	return &locksCollector{
		locks: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, locksSubsystem, "table"),
			"Number of locks by datname, locktype, mode and granted",
			[]string{"datname", "locktype", "mode", "granted"},
			nil,
		),
	}, nil
}

func (c *locksCollector) Update(ctx context.Context, db *pgx.Conn, ch chan<- prometheus.Metric) error {
	rows, err := db.QueryEx(ctx, locksQuery, nil)
	if err != nil {
		return err
	}

	defer rows.Close()

	var datname, locktype, mode string
	var granted bool
	var count float64

	for rows.Next() {
		if err := rows.Scan(&datname, &locktype, &mode, &granted, &count); err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(
			c.locks,
			prometheus.GaugeValue,
			count,
			datname,
			locktype,
			mode,
			strconv.FormatBool(granted),
		)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
