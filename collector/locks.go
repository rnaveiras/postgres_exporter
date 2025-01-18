package collector

import (
	"context"
	"strconv"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	locksQuery = `
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

type locksScraper struct {
	locks *prometheus.Desc
}

// NewLocksScraper returns a new Scraper exposing data from pg_locks
func NewLocksScraper() Scraper {
	return &locksScraper{
		locks: prometheus.NewDesc(
			"postgres_locks_table",
			"Number of locks by datname, locktype, mode and granted",
			[]string{"datname", "locktype", "mode", "granted"},
			nil,
		),
	}
}

func (*locksScraper) Name() string {
	return "LocksScraper"
}

func (c *locksScraper) Scrape(ctx context.Context, conn *pgx.Conn, _ Version, ch chan<- prometheus.Metric) error {
	rows, err := conn.Query(ctx, locksQuery)
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
