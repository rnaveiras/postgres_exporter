package collector

import (
	"context"
	"time"

	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
)

// The Statistics Scraper
// PostgreSQL's statistics collector is a subsystem that supports collection and reporting of information about
// server activity. Presently, the collector can count accesses to tables and indexes in both disk-block and
// individual-row terms. It also tracks the total number of rows in each table, and information about vacuum
// and analyze actions for each table. It can also count calls to user-defined functions and the total time
// spent in each one.

//https://www.postgresql.org/docs/9.4/static/monitoring-stats.html#PG-STAT-ALL-TABLES-VIEW

const (
	// Scrape query
	statUserTablesQuery = `
SELECT schemaname
     , relname
     , seq_scan::float
     , seq_tup_read::float
     , idx_scan::float
     , idx_tup_fetch::float
     , n_tup_ins::float
     , n_tup_upd::float
     , n_tup_del::float
     , n_tup_hot_upd::float
     , n_live_tup::float
     , n_dead_tup::float
     , n_mod_since_analyze::float
     , COALESCE(last_analyze, make_timestamptz(1970,01,01,0,0,0.0,'UTC')) AS last_analyze
     , COALESCE(last_autoanalyze, make_timestamptz(1970,01,01,0,0,0.0,'UTC')) AS last_autoanalyze
     , COALESCE(last_vacuum, make_timestamptz(1970,01,01,0,0,0.0,'UTC')) AS last_vacuum
     , COALESCE(last_autovacuum, make_timestamptz(1970,01,01,0,0,0.0,'UTC')) AS last_autovacuum
     , vacuum_count::float
     , autovacuum_count::float
     , analyze_count::float
     , autoanalyze_count::float
  FROM pg_stat_user_tables
 WHERE schemaname != 'information_schema' /*postgres_exporter*/`
)

type statUserTablesScraper struct {
	seqScan          *prometheus.Desc
	seqTupRead       *prometheus.Desc
	idxScan          *prometheus.Desc
	idxTupFetch      *prometheus.Desc
	nTupIns          *prometheus.Desc
	nTupUpd          *prometheus.Desc
	nTupDel          *prometheus.Desc
	nTupHotUpd       *prometheus.Desc
	nLiveTup         *prometheus.Desc
	nDeadTup         *prometheus.Desc
	nModSinceAnalyze *prometheus.Desc
	lastAnalyze      *prometheus.Desc
	lastAutoAnalyze  *prometheus.Desc
	lastVacuum       *prometheus.Desc
	lastAutoVacuum   *prometheus.Desc
	vacuumCount      *prometheus.Desc
	autovacuumCount  *prometheus.Desc
	analyzeCount     *prometheus.Desc
	autoanalyzeCount *prometheus.Desc
}

// NewStatUserTablesScraper returns a new Scraper exposing postgres pg_stat_database view
func NewStatUserTablesScraper() Scraper {
	return &statUserTablesScraper{
		seqScan: prometheus.NewDesc(
			"postgres_stat_user_tablesseq_scan_total",
			"Number of sequential scans initiated on this table",
			[]string{"schemaname", "relname"},
			nil,
		),
		seqTupRead: prometheus.NewDesc(
			"postgres_stat_user_tablesseq_tup_read_total",
			"Number of live rows fetched by sequential scans",
			[]string{"schemaname", "relname"},
			nil,
		),
		idxScan: prometheus.NewDesc(
			"postgres_stat_user_tablesidx_scan_total",
			"Number of index scans initiated on this table",
			[]string{"schemaname", "relname"},
			nil,
		),
		idxTupFetch: prometheus.NewDesc(
			"postgres_stat_user_tablesidx_tup_fetch_total",
			"Number of live rows fetched by index scans",
			[]string{"schemaname", "relname"},
			nil,
		),
		nTupIns: prometheus.NewDesc(
			"postgres_stat_user_tablesn_tup_ins_total",
			"Number of rows inserted",
			[]string{"schemaname", "relname"},
			nil,
		),
		nTupUpd: prometheus.NewDesc(
			"postgres_stat_user_tablesn_tup_upd_total",
			"Number of rows updated",
			[]string{"schemaname", "relname"},
			nil,
		),
		nTupDel: prometheus.NewDesc(
			"postgres_stat_user_tablesn_tup_del_total",
			"Number of rows deleted",
			[]string{"schemaname", "relname"},
			nil,
		),
		nTupHotUpd: prometheus.NewDesc(
			"postgres_stat_user_tablesn_tup_hot_upd",
			"Number of rows HOT updated (i.e., with no separate index update required)",
			[]string{"schemaname", "relname"},
			nil,
		),
		nLiveTup: prometheus.NewDesc(
			"postgres_stat_user_tablesn_live_tup",
			"Estimated number of live rows",
			[]string{"schemaname", "relname"},
			nil,
		),
		nDeadTup: prometheus.NewDesc(
			"postgres_stat_user_tablesn_dead_tup",
			"Estimated number of dead rows",
			[]string{"schemaname", "relname"},
			nil,
		),
		nModSinceAnalyze: prometheus.NewDesc(
			"postgres_stat_user_tablesn_mod_since_analyze",
			"Estimated number of rows modified since this table was last analyzed",
			[]string{"schemaname", "relname"},
			nil,
		),
		lastAnalyze: prometheus.NewDesc(
			"postgres_stat_user_tableslast_analyze_timestamp",
			"Last time at which this table was manually analyzed",
			[]string{"schemaname", "relname"},
			nil,
		),
		lastAutoAnalyze: prometheus.NewDesc(
			"postgres_stat_user_tableslast_autoanalyze_timestamp",
			"Last time at which this table was analyzed by the autovacuum daemon",
			[]string{"schemaname", "relname"},
			nil,
		),
		lastVacuum: prometheus.NewDesc(
			"postgres_stat_user_tableslast_vacuum_timestamp",
			"Last time at which this table was manually vacuumed (not counting VACUUM FULL)",
			[]string{"schemaname", "relname"},
			nil,
		),
		lastAutoVacuum: prometheus.NewDesc(
			"postgres_stat_user_tableslast_autovacuum_timestamp",
			"Last time at which this table was vacuumed by the autovacuum daemon",
			[]string{"schemaname", "relname"},
			nil,
		),
		vacuumCount: prometheus.NewDesc(
			"postgres_stat_user_tablesvacuum_total",
			"Number of times this table has been manually vacuumed (not counting VACUUM FULL)",
			[]string{"schemaname", "relname"},
			nil,
		),
		autovacuumCount: prometheus.NewDesc(
			"postgres_stat_user_tables_autovacuum_total",
			"Number of times this table has been vacuumed by the autovacuum daemon",
			[]string{"schemaname", "relname"},
			nil,
		),
		analyzeCount: prometheus.NewDesc(
			"postgres_stat_user_tables_analyze_total",
			"Number of times this table has been manually analyzed",
			[]string{"schemaname", "relname"},
			nil,
		),
		autoanalyzeCount: prometheus.NewDesc(
			"postgres_stat_user_tables_autoanalyze_total",
			"Number of times this table has been analyzed by the autovacuum daemon",
			[]string{"schemaname", "relname"},
			nil,
		),
	}
}

func (c *statUserTablesScraper) Name() string {
	return "StatUserTablesScraper"
}

func (c *statUserTablesScraper) Scrape(ctx context.Context, conn *pgx.Conn, ch chan<- prometheus.Metric) error {
	rows, err := conn.QueryEx(ctx, statUserTablesQuery, nil)
	if err != nil {
		return err
	}
	defer rows.Close()

	var schemaname, relname string
	var seqScan, seqTupRead, idxScan, idxTupFetch, nTupIns, nTupUpd, nTupDel,
		nTupHotUpd, nLiveTup, nDeadTup, nModSinceAnalyze, vacuumCount,
		autovacuumCount, analyzeCount, autoanalyzeCount float64
	var lastAnalyze, lastVacuum, lastAutoAnalyze, lastAutoVacuum time.Time
	for rows.Next() {
		if err := rows.Scan(&schemaname,
			&relname,
			&seqScan,
			&seqTupRead,
			&idxScan,
			&idxTupFetch,
			&nTupIns,
			&nTupUpd,
			&nTupDel,
			&nTupHotUpd,
			&nLiveTup,
			&nDeadTup,
			&nModSinceAnalyze,
			&lastAnalyze,
			&lastAutoAnalyze,
			&lastVacuum,
			&lastAutoVacuum,
			&vacuumCount,
			&autovacuumCount,
			&analyzeCount,
			&autoanalyzeCount); err != nil {
			return err
		}

		// postgres_stat_user_tables_seq_scan
		ch <- prometheus.MustNewConstMetric(c.seqScan, prometheus.CounterValue, seqScan, schemaname, relname)
		// postgres_stat_user_tables_seq_tup_read
		ch <- prometheus.MustNewConstMetric(c.seqTupRead, prometheus.CounterValue, seqTupRead, schemaname, relname)

		// postgres_stat_user_tables_idx_scan_total
		ch <- prometheus.MustNewConstMetric(c.idxScan, prometheus.CounterValue, idxScan, schemaname, relname)
		// postgres_stat_user_tables_idx_fetch_total
		ch <- prometheus.MustNewConstMetric(c.idxTupFetch, prometheus.CounterValue, idxTupFetch, schemaname, relname)

		// postgres_stat_user_tables_n_tup_in_total
		ch <- prometheus.MustNewConstMetric(c.nTupIns, prometheus.CounterValue, nTupIns, schemaname, relname)
		// postgres_stat_user_tables_n_tup_upd_total
		ch <- prometheus.MustNewConstMetric(c.nTupUpd, prometheus.CounterValue, nTupUpd, schemaname, relname)
		// postgres_stat_user_tables_n_tup_del_total
		ch <- prometheus.MustNewConstMetric(c.nTupDel, prometheus.CounterValue, nTupDel, schemaname, relname)
		// postgres_stat_user_tables_n_tup_hot_upd_total
		ch <- prometheus.MustNewConstMetric(c.nTupHotUpd, prometheus.CounterValue, nTupHotUpd, schemaname, relname)

		// postgres_stat_user_tables_n_live_tup
		ch <- prometheus.MustNewConstMetric(c.nLiveTup, prometheus.GaugeValue, nLiveTup, schemaname, relname)
		// postgres_stat_user_tables_n_dead_tup
		ch <- prometheus.MustNewConstMetric(c.nDeadTup, prometheus.GaugeValue, nDeadTup, schemaname, relname)
		// postgres_stat_user_tables_n_mod_since_analyze
		ch <- prometheus.MustNewConstMetric(c.nModSinceAnalyze, prometheus.GaugeValue, nModSinceAnalyze, schemaname, relname)

		// postgres_stat_user_tables_last_analyze_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastAnalyze, prometheus.GaugeValue, float64(lastAnalyze.UTC().Unix()), schemaname, relname)
		// postgres_stat_user_tables_last_autoanalyze_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastAutoAnalyze, prometheus.GaugeValue, float64(lastAutoAnalyze.UTC().Unix()), schemaname, relname)
		// postgres_stat_user_tables_last_vacuum_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastVacuum, prometheus.GaugeValue, float64(lastVacuum.UTC().Unix()), schemaname, relname)
		// postgres_stat_user_tables_last_autovacuum_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastAutoVacuum, prometheus.GaugeValue, float64(lastAutoVacuum.UTC().Unix()), schemaname, relname)

		// postgres_stat_user_tables_vacuum_total
		ch <- prometheus.MustNewConstMetric(c.vacuumCount, prometheus.CounterValue, vacuumCount, schemaname, relname)
		// postgres_stat_user_tables_autovacuum_total
		ch <- prometheus.MustNewConstMetric(c.autovacuumCount, prometheus.CounterValue, autovacuumCount, schemaname, relname)
		// postgres_stat_user_tables_analyze_total
		ch <- prometheus.MustNewConstMetric(c.analyzeCount, prometheus.CounterValue, analyzeCount, schemaname, relname)
		// postgres_stat_user_tables_autovacuum_total
		ch <- prometheus.MustNewConstMetric(c.autoanalyzeCount, prometheus.CounterValue, autoanalyzeCount, schemaname, relname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
