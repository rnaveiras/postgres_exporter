package collector

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Subsystem
	statReplicationSubsystem = "stat_replication"

	// Scrape query
	statReplicatonLagBytes = `
SELECT application_name
     , client_addr
     , state
     , sync_state
     , pg_xlog_location_diff(pg_current_xlog_location(), replay_location)::float AS pg_xlog_location_diff
  FROM pg_stat_replication /*postgres_exporter*/`
)

type statReplicationCollector struct {
	lagBytes *prometheus.Desc
}

func init() {
	registerCollector("stat_replication", defaultEnabled, NewStatReplicationCollector)
}

// NewStatReplicationCollector returns a new Collector exposing postgres pg_stat_replication
func NewStatReplicationCollector() (Collector, error) {
	return &statReplicationCollector{
		lagBytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statReplicationSubsystem, "lag_bytes"),
			"delay in bytes pg_xlog_location_diff(pg_current_xlog_location(), replay_location)",
			[]string{"application_name", "client_addr", "state", "sync_state"},
			nil,
		),
	}, nil
}

func (c *statReplicationCollector) Update(ctx context.Context, db *pgx.Conn, ch chan<- prometheus.Metric) error {
	rows, err := db.QueryEx(ctx, statReplicatonLagBytes, nil)
	if err != nil {
		return err
	}
	defer rows.Close()

	var applicationName, clientAddr, state, syncState string
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
			clientAddr,
			state,
			syncState)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
