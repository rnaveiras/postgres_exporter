package collector

import (
	"context"
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// The Statistics Collector
// PostgreSQL's statistics collector is a subsystem that supports collection and reporting of information about
// server activity. Presently, the collector can count accesses to tables and indexes in both disk-block and
// individual-row terms. It also tracks the total number of rows in each table, and information about vacuum
// and analyze actions for each table. It can also count calls to user-defined functions and the total time
// spent in each one.

//https://www.postgresql.org/docs/9.4/static/monitoring-stats.html#PG-STAT-ALL-TABLES-VIEW

const (
	// Subsystem
	statUserTablesSubsystem = "stat_user_tables"
	// Scrape query
	statUserTablesQuery = `SELECT schemaname, relname, seq_scan, last_analyze FROM pg_all_user_tables`
)

type statUserTablesCollector struct {
	seqScan     *prometheus.Desc
	lastAnalyze *prometheus.Desc
}

func init() {
	registerCollector("stat_user_tables", defaultDisabled, NewStatUserTablesCollector)
}

// NewStatUserTablesCollector returns a new Collector exposing postgres pg_stat_database view
func NewStatUserTablesCollector() (Collector, error) {
	return &statUserTablesCollector{
		seqScan: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statUserTablesSubsystem, "seq_scan_total"),
			"Number of sequential scans initiated on this table",
			[]string{"schemaname", "relname"},
			nil,
		),
		lastAnalyze: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, statUserTablesSubsystem, "timestamp_last_analyze_seconds"),
			"Last time at which this table was manually analyzed",
			[]string{"schemaname", "relname"},
			nil,
		),
	}, nil
}

func (c *statUserTablesCollector) Update(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.QueryContext(ctx, statUserTablesQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var schemaname, relname string
	var seqScan float64
	var lastAnalyze time.Time
	for rows.Next() {
		if err := rows.Scan(&schemaname, &relname, &seqScan, &lastAnalyze); err != nil {
			return err
		}

		log.Infoln("analyze", lastAnalyze)
		// postgres_stat_user_tables_seq_scan
		ch <- prometheus.MustNewConstMetric(c.seqScan, prometheus.CounterValue, seqScan, schemaname, relname)
		// postgres_stat_user_tables_timestamp_last_analyze_seconds
		ch <- prometheus.MustNewConstMetric(c.lastAnalyze, prometheus.GaugeValue, float64(lastAnalyze.UTC().Unix()), schemaname, relname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
