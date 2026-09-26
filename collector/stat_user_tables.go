package collector

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// The Statistics Scraper
// PostgreSQL's statistics collector is a subsystem that supports collection and reporting of information about
// server activity. Presently, the collector can count accesses to tables and indexes in both disk-block and
// individual-row terms. It also tracks the total number of rows in each table, and information about vacuum
// and analyze actions for each table. It can also count calls to user-defined functions and the total time
// spent in each one.
// https://www.postgresql.org/docs/9.4/static/monitoring-stats.html#PG-STAT-ALL-TABLES-VIEW
const (
	// Scrape query.
	statUserTablesQuery = `
SELECT schemaname
     , relname
     , seq_scan::float
     , seq_tup_read::float
     , COALESCE(idx_scan, 0)::float AS idx_scan
     , COALESCE(idx_tup_fetch, 0)::float AS idx_tup_fetch
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
	nTupHotUpdLegacy *prometheus.Desc
	legacyNames      bool
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

// NewStatUserTablesScraper returns a new Scraper exposing postgres pg_stat_user_tables view. With legacyNames
// it also emits the deprecated metric names.
func NewStatUserTablesScraper(legacyNames bool) Scraper {
	return &statUserTablesScraper{
		legacyNames: legacyNames,
		seqScan: prometheus.NewDesc(
			"postgres_stat_user_tables_seq_scan_total",
			"Number of sequential scans initiated on this table",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		seqTupRead: prometheus.NewDesc(
			"postgres_stat_user_tables_seq_tup_read_total",
			"Number of live rows fetched by sequential scans",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		idxScan: prometheus.NewDesc(
			"postgres_stat_user_tables_idx_scan_total",
			"Number of index scans initiated on this table",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		idxTupFetch: prometheus.NewDesc(
			"postgres_stat_user_tables_idx_tup_fetch_total",
			"Number of live rows fetched by index scans",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nTupIns: prometheus.NewDesc(
			"postgres_stat_user_tables_n_tup_ins_total",
			"Number of rows inserted",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nTupUpd: prometheus.NewDesc(
			"postgres_stat_user_tables_n_tup_upd_total",
			"Number of rows updated",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nTupDel: prometheus.NewDesc(
			"postgres_stat_user_tables_n_tup_del_total",
			"Number of rows deleted",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nTupHotUpd: prometheus.NewDesc(
			"postgres_stat_user_tables_n_tup_hot_upd_total",
			"Number of rows HOT updated (i.e., with no separate index update required)",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nTupHotUpdLegacy: prometheus.NewDesc(
			"postgres_stat_user_tables_n_tup_hot_upd",
			"(DEPRECATED) Use postgres_stat_user_tables_n_tup_hot_upd_total. Number of rows HOT updated (i.e., with no separate index update required)",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nLiveTup: prometheus.NewDesc(
			"postgres_stat_user_tables_n_live_tup",
			"Estimated number of live rows",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nDeadTup: prometheus.NewDesc(
			"postgres_stat_user_tables_n_dead_tup",
			"Estimated number of dead rows",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		nModSinceAnalyze: prometheus.NewDesc(
			"postgres_stat_user_tables_n_mod_since_analyze",
			"Estimated number of rows modified since this table was last analyzed",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		lastAnalyze: prometheus.NewDesc(
			"postgres_stat_user_tables_last_analyze_timestamp",
			"Last time at which this table was manually analyzed",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		lastAutoAnalyze: prometheus.NewDesc(
			"postgres_stat_user_tables_last_autoanalyze_timestamp",
			"Last time at which this table was analyzed by the autovacuum daemon",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		lastVacuum: prometheus.NewDesc(
			"postgres_stat_user_tables_last_vacuum_timestamp",
			"Last time at which this table was manually vacuumed (not counting VACUUM FULL)",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		lastAutoVacuum: prometheus.NewDesc(
			"postgres_stat_user_tables_last_autovacuum_timestamp",
			"Last time at which this table was vacuumed by the autovacuum daemon",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		vacuumCount: prometheus.NewDesc(
			"postgres_stat_user_tables_vacuum_total",
			"Number of times this table has been manually vacuumed (not counting VACUUM FULL)",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		autovacuumCount: prometheus.NewDesc(
			"postgres_stat_user_tables_autovacuum_total",
			"Number of times this table has been vacuumed by the autovacuum daemon",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		analyzeCount: prometheus.NewDesc(
			"postgres_stat_user_tables_analyze_total",
			"Number of times this table has been manually analyzed",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
		autoanalyzeCount: prometheus.NewDesc(
			"postgres_stat_user_tables_autoanalyze_total",
			"Number of times this table has been analyzed by the autovacuum daemon",
			[]string{labelDatname, labelSchemaname, labelRelname},
			nil,
		),
	}
}

func (*statUserTablesScraper) Name() string {
	return "StatUserTablesScraper"
}

func (c *statUserTablesScraper) Scrape(ctx context.Context, conn Querier, _ Version, ch chan<- prometheus.Metric) error {
	var datname string
	if err := conn.QueryRow(ctx, currentDatabaseQuery).Scan(&datname); err != nil {
		return err
	}

	rows, err := conn.Query(ctx, statUserTablesQuery)
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
		ch <- prometheus.MustNewConstMetric(c.seqScan, prometheus.CounterValue, seqScan, datname, schemaname, relname)
		// postgres_stat_user_tables_seq_tup_read
		ch <- prometheus.MustNewConstMetric(c.seqTupRead, prometheus.CounterValue, seqTupRead, datname, schemaname, relname)

		// postgres_stat_user_tables_idx_scan_total
		ch <- prometheus.MustNewConstMetric(c.idxScan, prometheus.CounterValue, idxScan, datname, schemaname, relname)
		// postgres_stat_user_tables_idx_fetch_total
		ch <- prometheus.MustNewConstMetric(c.idxTupFetch, prometheus.CounterValue, idxTupFetch, datname, schemaname, relname)

		// postgres_stat_user_tables_n_tup_in_total
		ch <- prometheus.MustNewConstMetric(c.nTupIns, prometheus.CounterValue, nTupIns, datname, schemaname, relname)
		// postgres_stat_user_tables_n_tup_upd_total
		ch <- prometheus.MustNewConstMetric(c.nTupUpd, prometheus.CounterValue, nTupUpd, datname, schemaname, relname)
		// postgres_stat_user_tables_n_tup_del_total
		ch <- prometheus.MustNewConstMetric(c.nTupDel, prometheus.CounterValue, nTupDel, datname, schemaname, relname)
		// postgres_stat_user_tables_n_tup_hot_upd_total
		ch <- prometheus.MustNewConstMetric(c.nTupHotUpd, prometheus.CounterValue, nTupHotUpd, datname, schemaname, relname)
		if c.legacyNames {
			// postgres_stat_user_tables_n_tup_hot_upd (deprecated)
			ch <- prometheus.MustNewConstMetric(c.nTupHotUpdLegacy, prometheus.CounterValue, nTupHotUpd, datname, schemaname, relname)
		}

		// postgres_stat_user_tables_n_live_tup
		ch <- prometheus.MustNewConstMetric(c.nLiveTup, prometheus.GaugeValue, nLiveTup, datname, schemaname, relname)
		// postgres_stat_user_tables_n_dead_tup
		ch <- prometheus.MustNewConstMetric(c.nDeadTup, prometheus.GaugeValue, nDeadTup, datname, schemaname, relname)
		// postgres_stat_user_tables_n_mod_since_analyze
		ch <- prometheus.MustNewConstMetric(c.nModSinceAnalyze, prometheus.GaugeValue, nModSinceAnalyze, datname, schemaname, relname)

		// postgres_stat_user_tables_last_analyze_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastAnalyze, prometheus.GaugeValue, float64(lastAnalyze.UTC().Unix()), datname, schemaname, relname)
		// postgres_stat_user_tables_last_autoanalyze_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastAutoAnalyze, prometheus.GaugeValue, float64(lastAutoAnalyze.UTC().Unix()), datname, schemaname, relname)
		// postgres_stat_user_tables_last_vacuum_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastVacuum, prometheus.GaugeValue, float64(lastVacuum.UTC().Unix()), datname, schemaname, relname)
		// postgres_stat_user_tables_last_autovacuum_timestamp
		ch <- prometheus.MustNewConstMetric(c.lastAutoVacuum, prometheus.GaugeValue, float64(lastAutoVacuum.UTC().Unix()), datname, schemaname, relname)

		// postgres_stat_user_tables_vacuum_total
		ch <- prometheus.MustNewConstMetric(c.vacuumCount, prometheus.CounterValue, vacuumCount, datname, schemaname, relname)
		// postgres_stat_user_tables_autovacuum_total
		ch <- prometheus.MustNewConstMetric(c.autovacuumCount, prometheus.CounterValue, autovacuumCount, datname, schemaname, relname)
		// postgres_stat_user_tables_analyze_total
		ch <- prometheus.MustNewConstMetric(c.analyzeCount, prometheus.CounterValue, analyzeCount, datname, schemaname, relname)
		// postgres_stat_user_tables_autovacuum_total
		ch <- prometheus.MustNewConstMetric(c.autoanalyzeCount, prometheus.CounterValue, autoanalyzeCount, datname, schemaname, relname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
