package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	statDatabaseQuery = `
SELECT datname
     , numbackends::float
     , tup_returned::float
     , tup_fetched::float
     , tup_inserted::float
     , tup_updated::float
     , tup_deleted::float
     , xact_commit::float
     , xact_rollback::float
     , blks_read::float
     , blks_hit::float
     , conflicts::float
     , deadlocks::float
     , temp_files::float
     , temp_bytes::float
  FROM pg_stat_database
  WHERE datname IS NOT NULL /*postgres_exporter*/`
)

type statDatabaseScraper struct {
	numbackends  *prometheus.Desc
	tupReturned  *prometheus.Desc
	tupFetched   *prometheus.Desc
	tupInserted  *prometheus.Desc
	tupUpdated   *prometheus.Desc
	tupDeleted   *prometheus.Desc
	xactCommit   *prometheus.Desc
	xactRollback *prometheus.Desc
	blksRead     *prometheus.Desc
	blksHit      *prometheus.Desc
	conflicts    *prometheus.Desc
	deadlocks    *prometheus.Desc
	tempFiles    *prometheus.Desc
	tempBytes    *prometheus.Desc
}

// NewStatDatabaseScraper returns a new Scraper exposing postgres pg_stat_database view
// The Statistics Scraper
// PostgreSQL's statistics collector is a subsystem that supports collection and reporting of information about
// server activity. Presently, the collector can count accesses to tables and indexes in both disk-block and
// individual-row terms. It also tracks the total number of rows in each table, and information about vacuum
// and analyze actions for each table. It can also count calls to user-defined functions and the total time
// spent in each one.
// https://www.postgresql.org/docs/9.4/static/monitoring-stats.html#PG-STAT-DATABASE-VIEW
func NewStatDatabaseScraper() Scraper {
	return &statDatabaseScraper{
		numbackends: prometheus.NewDesc(
			"postgres_stat_database_numbackends",
			"Number of backends currently connected to this database. This is the only column in this"+
				" view that returns a value reflecting current state; all other columns return the accumulated"+
				" values since the last reset.",
			[]string{"datname"},
			nil,
		),
		tupReturned: prometheus.NewDesc(
			"postgres_stat_database_tup_returned_total",
			"Number of rows returned by queries in this database",
			[]string{"datname"},
			nil,
		),
		tupFetched: prometheus.NewDesc(
			"postgres_stat_database_tup_fetched_total",
			"Number of rows fetched by queries in this database",
			[]string{"datname"},
			nil,
		),
		tupInserted: prometheus.NewDesc(
			"postgres_stat_database_tup_inserted_total",
			"Number of rows inserted by queries in this database",
			[]string{"datname"},
			nil,
		),
		tupUpdated: prometheus.NewDesc(
			"postgres_stat_database_tup_updated_total",
			"Number of rows updated by queries in this database",
			[]string{"datname"},
			nil,
		),
		tupDeleted: prometheus.NewDesc(
			"postgres_stat_database_tup_deleted_total",
			"Number of rows deleted by queries in this database",
			[]string{"datname"},
			nil,
		),
		xactCommit: prometheus.NewDesc(
			"postgres_stat_database_xact_commit_total",
			"Number of transactions in this database that have been committed",
			[]string{"datname"},
			nil,
		),
		xactRollback: prometheus.NewDesc(
			"postgres_stat_database_xact_rollback_total",
			"Number of transactions in this database that have been rolled back",
			[]string{"datname"},
			nil,
		),
		blksRead: prometheus.NewDesc(
			"postgres_stat_database_blks_read_total",
			"Number of disk blocks read in this database",
			[]string{"datname"},
			nil,
		),
		blksHit: prometheus.NewDesc(
			"postgres_stat_database_blks_hit_total",
			"Number of times disk blocks were found already in the buffer cache, so that a read was not necessary"+
				" (this only includes hits in the PostgreSQL buffer cache, not the operating system's file system cache)",
			[]string{"datname"},
			nil,
		),
		conflicts: prometheus.NewDesc(
			"postgres_stat_database_conflicts_total",
			"Number of queries canceled due to conflicts with recovery in this database."+
				" (Conflicts occur only on standby servers; see pg_stat_database_conflicts for details.)",
			[]string{"datname"},
			nil,
		),
		deadlocks: prometheus.NewDesc(
			"postgres_stat_database_deadlocks_total",
			"Number of deadlocks detected in this database",
			[]string{"datname"},
			nil,
		),
		tempFiles: prometheus.NewDesc(
			"postgres_stat_database_temp_files_total",
			"Number of temporary files created by queries in this database. All temporary files are counted,"+
				" regardless of why the temporary file was created (e.g., sorting or hashing), and regardless of "+
				" the log_temp_files setting.",
			[]string{"datname"},
			nil,
		),
		tempBytes: prometheus.NewDesc(
			"postgres_stat_database_temp_bytes_total",
			"Total amount of data written to temporary files by queries in this database. All temporary files"+
				" are counted, regardless of why the temporary file was created, and regardless of the log_temp_files setting.",
			[]string{"datname"},
			nil,
		),
	}
}

func (*statDatabaseScraper) Name() string {
	return "StatDatabaseScraper"
}

func (c *statDatabaseScraper) Scrape(ctx context.Context, conn *pgx.Conn, _ Version, ch chan<- prometheus.Metric) error {
	rows, err := conn.Query(ctx, statDatabaseQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	type databaseRow struct {
		datname      string
		numbackends  float64
		tupReturned  float64
		tupFetched   float64
		tupInserted  float64
		tupUpdated   float64
		tupDeleted   float64 // Used in metrics below
		xactCommit   float64
		xactRollback float64
		blksRead     float64
		blksHit      float64
		conflicts    float64
		deadlocks    float64
		tempFiles    float64
		tempBytes    float64
	}

	dbRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[databaseRow])
	if err != nil {
		return err
	}

	for _, row := range dbRows {
		// postgres_stat_database_numbackends
		ch <- prometheus.MustNewConstMetric(c.numbackends, prometheus.GaugeValue, row.numbackends, row.datname)
		// postgres_stat_database_tup_returned_total
		ch <- prometheus.MustNewConstMetric(c.tupReturned, prometheus.CounterValue, row.tupReturned, row.datname)
		// postgres_stat_database_tup_fetched_total
		ch <- prometheus.MustNewConstMetric(c.tupFetched, prometheus.CounterValue, row.tupFetched, row.datname)
		// postgres_stat_database_tup_inserted_total
		ch <- prometheus.MustNewConstMetric(c.tupInserted, prometheus.CounterValue, row.tupInserted, row.datname)
		// postgres_stat_database_tup_updated_total
		ch <- prometheus.MustNewConstMetric(c.tupUpdated, prometheus.CounterValue, row.tupUpdated, row.datname)
		// postgres_stat_database_tup_deleted_total
		ch <- prometheus.MustNewConstMetric(c.tupDeleted, prometheus.CounterValue, row.tupDeleted, row.datname)
		// postgres_stat_database_xact_commit_total
		ch <- prometheus.MustNewConstMetric(c.xactCommit, prometheus.CounterValue, row.xactCommit, row.datname)
		// postgres_stat_database_tup_xact_rollback_total
		ch <- prometheus.MustNewConstMetric(c.xactRollback, prometheus.CounterValue, row.xactRollback, row.datname)
		// postgres_stat_database_blks_read_total
		ch <- prometheus.MustNewConstMetric(c.blksRead, prometheus.CounterValue, row.blksRead, row.datname)
		// postgres_stat_database_blks_hit_total
		ch <- prometheus.MustNewConstMetric(c.blksHit, prometheus.CounterValue, row.blksHit, row.datname)
		// postgres_stat_database_conflicts_total
		ch <- prometheus.MustNewConstMetric(c.conflicts, prometheus.CounterValue, row.conflicts, row.datname)
		// postgres_stat_database_deadlocks_total
		ch <- prometheus.MustNewConstMetric(c.deadlocks, prometheus.CounterValue, row.deadlocks, row.datname)
		// postgres_stat_database_temp_files_total
		ch <- prometheus.MustNewConstMetric(c.tempFiles, prometheus.CounterValue, row.tempFiles, row.datname)
		// postgres_stat_database_temp_bytes_total
		ch <- prometheus.MustNewConstMetric(c.tempBytes, prometheus.CounterValue, row.tempBytes, row.datname)
	}

	return nil
}
