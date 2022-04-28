package collector

import (
	"context"
	"database/sql"
	"net"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

/* When pg_basebackup is running in stream mode, it opens a second connection
to the server and starts streaming the transaction log in parallel while
running the backup. In both connections (state=backup and state=streaming) the
pg_log_location_diff is null and it requires to be excluded */
const (
	// Scrape query
	statReplicationLag = `
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
	   , EXTRACT (EPOCH FROM write_lag) as write_lag_seconds
	   , EXTRACT (EPOCH FROM flush_lag) as flush_lag_seconds
	   , EXTRACT (EPOCH FROM replay_lag) as replay_lag_seconds
    FROM pg_stat_replication
)
SELECT * FROM pg_replication WHERE pg_xlog_location_diff IS NOT NULL /*postgres_exporter*/`
)

type statReplicationScraper struct {
	lagBytes  *prometheus.Desc
	writeLag  *prometheus.Desc
	flushLag  *prometheus.Desc
	replayLag *prometheus.Desc
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
		writeLag: prometheus.NewDesc(
			"postgres_stat_replication_write_lag_seconds",
			"write_lag as reported by the pg_stat_replication view converted to seconds",
			[]string{"application_name", "client_addr", "state", "sync_state"},
			nil,
		),
		flushLag: prometheus.NewDesc(
			"postgres_stat_replication_flush_lag_seconds",
			"flush_lag as reported by the pg_stat_replication view converted to seconds",
			[]string{"application_name", "client_addr", "state", "sync_state"},
			nil,
		),
		replayLag: prometheus.NewDesc(
			"postgres_stat_replication_replay_lag_seconds",
			"replay_lag as reported by the pg_stat_replication view converted to seconds",
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

	rows, err = conn.Query(ctx, statReplicationLag)

	if err != nil {
		return err
	}
	defer rows.Close()

	var applicationName, state, syncState string
	var clientAddr net.IP
	var pgXlogLocationDiff float64
	/* When querying pg_stat_replication it may be that we don't have
	   values for the various lags as they vanish are a period of inactivity
	*/
	var writeLagSeconds, flushLagSeconds, replayLagSeconds sql.NullFloat64

	for rows.Next() {

		if err := rows.Scan(&applicationName,
			&clientAddr,
			&state,
			&syncState,
			&pgXlogLocationDiff,
			&writeLagSeconds,
			&flushLagSeconds,
			&replayLagSeconds); err != nil {

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

		if writeLagSeconds.Valid {
			// postgres_stat_replication_write_lag_seconds
			ch <- prometheus.MustNewConstMetric(c.writeLag,
				prometheus.GaugeValue,
				writeLagSeconds.Float64,
				applicationName,
				clientAddr.String(),
				state,
				syncState)
		}

		if flushLagSeconds.Valid {
			// postgres_stat_replication_flush_lag_seconds
			ch <- prometheus.MustNewConstMetric(c.flushLag,
				prometheus.GaugeValue,
				flushLagSeconds.Float64,
				applicationName,
				clientAddr.String(),
				state,
				syncState)
		}

		if replayLagSeconds.Valid {
			// postgres_stat_replication_replay_lag_seconds
			ch <- prometheus.MustNewConstMetric(c.replayLag,
				prometheus.GaugeValue,
				replayLagSeconds.Float64,
				applicationName,
				clientAddr.String(),
				state,
				syncState)
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
