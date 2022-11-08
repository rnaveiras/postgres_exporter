package collector

import (
	"context"
	"net"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

// When pg_basebackup is running in stream mode, it opens a second connection
// to the server and starts streaming the transaction log in parallel while
// running the backup. In both connections (state=backup and state=streaming) the
// pg_log_location_diff is null and it requires to be excluded
const (
	// Scrape query
	statReplicationLagBytes9x = `
WITH pg_replication AS (
  SELECT application_name
       , client_addr
       , state
       , sync_state
       , ( CASE when pg_is_in_recovery()
           THEN pg_xlog_location_diff(pg_last_xlog_receive_location(), replay_location)::float
           ELSE pg_xlog_location_diff(pg_current_xlog_location(), replay_location)::float
		   END
	     ) AS pg_xlog_location_diff
    FROM pg_stat_replication
)
SELECT * FROM pg_replication WHERE pg_xlog_location_diff IS NOT NULL /*postgres_exporter*/`

	statReplicationLagBytes = `
WITH pg_replication AS (
  SELECT application_name
       , client_addr
       , state
       , sync_state
       , ( CASE when pg_is_in_recovery()
           THEN pg_wal_lsn_diff(pg_last_wal_receive_lsn(), replay_lsn)::float
           ELSE pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)::float
		   END
	     ) AS pg_xlog_location_diff
    FROM pg_stat_replication
)
SELECT * FROM pg_replication WHERE pg_xlog_location_diff IS NOT NULL /*postgres_exporter*/`
)

type statReplicationScraper struct {
	lagBytes *prometheus.Desc
}

// NewStatReplicationScraper returns a new Scraper exposing postgres pg_stat_replication
func NewStatReplicationScraper() Scraper {
	return &statReplicationScraper{
		lagBytes: prometheus.NewDesc(
			"postgres_stat_replication_lag_bytes",
			"delay in bytes pg_wal_lsn_diff(pg_current_wal_lsn(), replay_location)",
			[]string{"application_name", "client_addr", "state", "sync_state"},
			nil,
		),
	}
}

func (c *statReplicationScraper) Name() string {
	return "StatReplicationScraper"
}

func (c *statReplicationScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	var rows pgx.Rows
	var err error

	if version.Gte(10) {
		rows, err = conn.Query(ctx, statReplicationLagBytes)
	} else {
		rows, err = conn.Query(ctx, statReplicationLagBytes9x)
	}

	if err != nil {
		return err
	}
	defer rows.Close()

	var applicationName, state, syncState string
	var clientAddr net.IP
	var pgXlogLocationDiff float64

	for rows.Next() {
		if err := rows.Scan(&applicationName,
			&clientAddr,
			&state,
			&syncState,
			&pgXlogLocationDiff); err != nil {

			return err
		}

		// postgres_stat_replication_lag_bytes
		ch <- prometheus.MustNewConstMetric(c.lagBytes,
			prometheus.GaugeValue,
			pgXlogLocationDiff,
			applicationName,
			clientAddr.String(),
			state,
			syncState)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
