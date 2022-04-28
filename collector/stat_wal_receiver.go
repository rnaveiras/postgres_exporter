package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

/* When pg_basebackup is running in stream mode, it opens a second connection
to the server and starts streaming the transaction log in parallel while
running the backup. In both connections (state=backup and state=streaming) the
pg_log_location_diff is null and it requires to be excluded */
const (
	// Scrape query
	statWalReceiver = `
WITH pg_wal_receiver AS (
  SELECT status
	   , ( 
		CASE WHEN pg_last_wal_receive_lsn() = pg_last_wal_replay_lsn()
		THEN 0
		ELSE EXTRACT (EPOCH FROM now() - pg_last_xact_replay_timestamp())
		END
	   ) as postgres_wal_receiver_replay_lag
    FROM pg_stat_wal_receiver
)
SELECT * FROM pg_wal_receiver WHERE postgres_wal_receiver_replay_lag IS NOT NULL /*postgres_exporter*/`
)

type statWalReceiverScraper struct {
	walReceiverReplayLag *prometheus.Desc
}

// NewStatWalReceiverScraper returns a new Scraper exposing postgres pg_stat_replication
func NewWalReceiverScraper() Scraper {
	return &statWalReceiverScraper{
		walReceiverReplayLag: prometheus.NewDesc(
			"postgres_wal_receiver_replay_lag_seconds",
			"delay in standby wal replay seconds EXTRACT (EPOCH FROM now() - pg_last_xact_replay_timestamp()",
			[]string{"status"},
			nil,
		),
	}
}

func (c *statWalReceiverScraper) Name() string {
	return "StatWalReceiverScraperr"
}

func (c *statWalReceiverScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	var rows pgx.Rows
	var err error

	rows, err = conn.Query(ctx, statWalReceiver)

	if err != nil {
		return err
	}
	defer rows.Close()

	var status string
	var pgWalReceiverReplayLag float64

	for rows.Next() {

		if err := rows.Scan(&status,
			&pgWalReceiverReplayLag); err != nil {

			return err
		}
		// postgres_wal_receiver_replay_lag_seconds
		ch <- prometheus.MustNewConstMetric(c.walReceiverReplayLag,
			prometheus.GaugeValue,
			pgWalReceiverReplayLag,
			status)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
