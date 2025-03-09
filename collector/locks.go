package collector

import (
	"context"
	"strconv"

	pgx "github.com/jackc/pgx/v5"
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

	type lockRow struct {
		datname  string
		locktype string
		mode     string
		granted  bool
		count    float64
	}

	lockRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[lockRow])
	if err != nil {
		return err
	}

	for _, row := range lockRows {
		ch <- prometheus.MustNewConstMetric(
			c.locks,
			prometheus.GaugeValue,
			row.count,
			row.datname,
			row.locktype,
			row.mode,
			strconv.FormatBool(row.granted),
		)
	}

	return nil
}
