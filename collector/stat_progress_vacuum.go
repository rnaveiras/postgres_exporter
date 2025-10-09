package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// metricEnabled represent the value used when a metrics state is
	// active/enabled
	metricEnabled = 1.0

	// Scrape query
	statVacuumProgress = `
SELECT V.pid::text
     , V.datname
     , T.schemaname
     , T.relname
	 , A.query_start::text
     , V.phase
     , V.heap_blks_total::float
     , V.heap_blks_scanned::float
     , V.heap_blks_vacuumed::float
     , V.index_vacuum_count::float
     , V.max_dead_tuples::float
     , V.num_dead_tuples::float
  FROM pg_stat_progress_vacuum AS V
   JOIN pg_stat_activity A ON (A.pid = V.pid)
   JOIN pg_stat_all_tables AS T ON (T.relid = V.relid)
 WHERE V.datname = current_database() /*postgres_exporter*/`
)

type statVacuumProgressScraper struct {
	running                     *prometheus.Desc
	phaseInitializing           *prometheus.Desc
	phaseScanningHeap           *prometheus.Desc
	phaseVacuumingIndexes       *prometheus.Desc
	phaseVacuumingHeap          *prometheus.Desc
	phaseCleaningUpIndexes      *prometheus.Desc
	phaseTruncatingHeap         *prometheus.Desc
	phasePerformingFinalCleanup *prometheus.Desc
	heapBlksTotal               *prometheus.Desc
	heapBlksScanned             *prometheus.Desc
	heapBlksVacuumed            *prometheus.Desc
	indexVacuumCount            *prometheus.Desc
	maxDeadTuples               *prometheus.Desc
	numDeadTuples               *prometheus.Desc
}

// NewStatVacuumProgressScraper returns a new Scraper exposing postgres pg_stat_vacuum_progress_*
func NewStatVacuumProgressScraper() Scraper {
	return &statVacuumProgressScraper{
		running: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_running",
			"VACUUM is running",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phaseInitializing: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_initializing",
			"VACUUM is preparing to begin scanning the heap",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phaseScanningHeap: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_scanning_heap",
			"VACUUM is currently scanning the heap",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phaseVacuumingIndexes: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_vacuuming_indexes",
			"VACUUM is currently vacuuming the indexes",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phaseVacuumingHeap: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_vacuuming_heap",
			"VACUUM is currently vacuuming the heap",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phaseCleaningUpIndexes: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_cleaning_up_indexes",
			"VACUUM is currently cleaning up indexes",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phaseTruncatingHeap: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_truncating_heap",
			"VACUUM is currently truncating the heap",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		phasePerformingFinalCleanup: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_performing_final_cleanup",
			"VACUUM is performing final cleanup",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		heapBlksTotal: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_heap_blks_total",
			"Total number of heap blocks in the table",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		heapBlksScanned: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_heap_blks_scanned",
			"Number of heap blocks scanned",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		heapBlksVacuumed: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_heap_blks_vacuumed",
			"Number of heap blocks vacuumed",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		indexVacuumCount: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_index_vacuum_count",
			"Number of completed index vacuum cycles",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		maxDeadTuples: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_max_dead_tuples",
			"Number of dead tuples that we can store before needing to perform an index vacuum cycle",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
		numDeadTuples: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_num_dead_tuples",
			"Number of dead tuples collected since the last index vacuum cycle",
			[]string{"pid", "query_start", "schemaname", "datname", "relname"},
			nil,
		),
	}
}

func (*statVacuumProgressScraper) Name() string {
	return "StatVacuumProgressScraper"
}

// emitPhaseMetric emits a Prometheus metric for the given vacuum progress phase.
// It maps PostgreSQL vacuum phases to corresponding phase-specific metrics.
func (c *statVacuumProgressScraper) emitPhaseMetric(phase, pid, queryStart, schemaname, datname, relname string, ch chan<- prometheus.Metric) {
	switch phase {
	case "initializing":
		ch <- prometheus.MustNewConstMetric(c.phaseInitializing, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	case "scanning heap":
		ch <- prometheus.MustNewConstMetric(c.phaseScanningHeap, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	case "vacuuming indexes":
		ch <- prometheus.MustNewConstMetric(c.phaseVacuumingIndexes, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	case "vacuuming heap":
		ch <- prometheus.MustNewConstMetric(c.phaseVacuumingHeap, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	case "cleaning up indexes":
		ch <- prometheus.MustNewConstMetric(c.phaseCleaningUpIndexes, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	case "truncating heap":
		ch <- prometheus.MustNewConstMetric(c.phaseTruncatingHeap, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	case "performing final cleanup":
		ch <- prometheus.MustNewConstMetric(c.phasePerformingFinalCleanup, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)
	default:
	}
}

func (c *statVacuumProgressScraper) Scrape(ctx context.Context, conn *pgx.Conn, _ Version, ch chan<- prometheus.Metric) error {
	rows, err := conn.Query(ctx, statVacuumProgress)
	if err != nil {
		return err
	}
	defer rows.Close()

	var pid, queryStart, schemaname, datname, relname, phase string
	var heapBlksTotal, heapBlksScanned, heapBlksVacuumed, indexVacuumCount, maxDeadTuples, numDeadTuples float64

	for rows.Next() {
		if err := rows.Scan(&pid,
			&datname,
			&schemaname,
			&relname,
			&queryStart,
			&phase,
			&heapBlksTotal,
			&heapBlksScanned,
			&heapBlksVacuumed,
			&indexVacuumCount,
			&maxDeadTuples,
			&numDeadTuples); err != nil {
			return err
		}

		// postgres_stat_vacuum_progress_running
		ch <- prometheus.MustNewConstMetric(c.running, prometheus.GaugeValue, metricEnabled,
			pid, queryStart, schemaname, datname, relname)

		c.emitPhaseMetric(phase, pid, queryStart, schemaname, datname, relname, ch)

		// postgres_stat_vacuum_progress_heap_blks_total
		ch <- prometheus.MustNewConstMetric(c.heapBlksTotal, prometheus.GaugeValue, heapBlksTotal, pid, queryStart, schemaname, datname, relname)

		// postgres_stat_vacuum_progress_heap_blks_scanned
		ch <- prometheus.MustNewConstMetric(c.heapBlksScanned, prometheus.GaugeValue, heapBlksScanned, pid, queryStart, schemaname, datname, relname)

		// postgres_stat_vacuum_progress_heap_blks_vacuumed
		ch <- prometheus.MustNewConstMetric(c.heapBlksVacuumed, prometheus.GaugeValue, heapBlksVacuumed, pid, queryStart, schemaname, datname, relname)

		// postgres_stat_vacuum_progress_index_vacuum_count
		ch <- prometheus.MustNewConstMetric(c.indexVacuumCount, prometheus.GaugeValue, indexVacuumCount, pid, queryStart, schemaname, datname, relname)

		// postgres_stat_vacuum_progress_max_dead_tuples
		ch <- prometheus.MustNewConstMetric(c.maxDeadTuples, prometheus.GaugeValue, maxDeadTuples, pid, queryStart, schemaname, datname, relname)

		// postgres_stat_vacuum_progress_num_dead_tuples
		ch <- prometheus.MustNewConstMetric(c.numDeadTuples, prometheus.GaugeValue, numDeadTuples, pid, queryStart, schemaname, datname, relname)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
