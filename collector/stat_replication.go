package collector

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus"
)

// When pg_basebackup is running in stream mode, it opens a second connection
// to the server and starts streaming the transaction log in parallel while
// running the backup. In both connections (state=backup and state=streaming) the
// lag is null and those rows are excluded.
//
// client_addr is NULL for replicas connected over a Unix socket; host() keeps it NULL and the label is empty.
const statReplicationLagBytes = `
WITH pg_replication AS (
  SELECT application_name
       , host(client_addr) AS client_addr
       , state
       , sync_state
       , ( CASE WHEN pg_is_in_recovery()
           THEN pg_wal_lsn_diff(pg_last_wal_receive_lsn(), replay_lsn)::float
           ELSE pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)::float
           END
         ) AS lag_bytes
    FROM pg_stat_replication
)
SELECT application_name, client_addr, state, sync_state, lag_bytes
  FROM pg_replication
 WHERE lag_bytes IS NOT NULL /*postgres_exporter*/`

type statReplicationScraper struct {
	lagBytes *prometheus.Desc
}

// NewStatReplicationScraper returns a new Scraper exposing postgres pg_stat_replication.
func NewStatReplicationScraper() Scraper {
	return &statReplicationScraper{
		lagBytes: prometheus.NewDesc(
			"postgres_stat_replication_lag_bytes",
			"Replication lag in bytes: pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn), or the receive position on a standby",
			[]string{"application_name", "client_addr", "state", "sync_state"},
			nil,
		),
	}
}

func (*statReplicationScraper) Name() string {
	return "StatReplicationScraper"
}

func (c *statReplicationScraper) Scrape(ctx context.Context, conn Querier, _ Version, ch chan<- prometheus.Metric) error {
	rows, err := conn.Query(ctx, statReplicationLagBytes)
	if err != nil {
		return err
	}
	defer rows.Close()

	var applicationName, state, syncState string
	var clientAddr pgtype.Text
	var lagBytes float64

	for rows.Next() {
		if err := rows.Scan(&applicationName,
			&clientAddr,
			&state,
			&syncState,
			&lagBytes); err != nil {
			return err
		}

		// postgres_stat_replication_lag_bytes
		ch <- prometheus.MustNewConstMetric(c.lagBytes,
			prometheus.GaugeValue,
			lagBytes,
			applicationName,
			clientAddr.String, // empty when NULL
			state,
			syncState)
	}

	return rows.Err()
}
