package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// metricEnabled represent the value used when a metrics state is
	// active/enabled.
	metricEnabled = 1.0

	// PostgreSQL 17 measures dead tuple storage in bytes and counts dead item identifiers instead of tuples.
	statVacuumProgressBytesVersion = 17

	statVacuumProgressPre17 = `
SELECT V.datname
     , T.schemaname
     , T.relname
     , V.phase
     , V.heap_blks_total::float
     , V.heap_blks_scanned::float
     , V.heap_blks_vacuumed::float
     , V.index_vacuum_count::float
     , V.max_dead_tuples::float
     , V.num_dead_tuples::float
  FROM pg_stat_progress_vacuum AS V
  JOIN pg_stat_all_tables AS T ON (T.relid = V.relid)
 WHERE V.datname = current_database() /*postgres_exporter*/`

	statVacuumProgress17 = `
SELECT V.datname
     , T.schemaname
     , T.relname
     , V.phase
     , V.heap_blks_total::float
     , V.heap_blks_scanned::float
     , V.heap_blks_vacuumed::float
     , V.index_vacuum_count::float
     , V.max_dead_tuple_bytes::float
     , V.dead_tuple_bytes::float
     , V.num_dead_item_ids::float
  FROM pg_stat_progress_vacuum AS V
  JOIN pg_stat_all_tables AS T ON (T.relid = V.relid)
 WHERE V.datname = current_database() /*postgres_exporter*/`
)

var statVacuumProgressQueries = versioned{
	bestEffortVersion:              statVacuumProgressPre17,
	statVacuumProgressBytesVersion: statVacuumProgress17,
}

// vacuumProgressLabels identify the table being vacuumed. Only one VACUUM can run on a table at a time.
var vacuumProgressLabels = []string{labelDatname, labelSchemaname, labelRelname}

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
	maxDeadTupleBytes           *prometheus.Desc
	deadTupleBytes              *prometheus.Desc
	numDeadItemIDs              *prometheus.Desc
}

// NewStatVacuumProgressScraper returns a new Scraper exposing postgres pg_stat_vacuum_progress_*.
func NewStatVacuumProgressScraper() Scraper {
	return &statVacuumProgressScraper{
		running: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_running",
			"VACUUM is running",
			vacuumProgressLabels,
			nil,
		),
		phaseInitializing: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_initializing",
			"VACUUM is preparing to begin scanning the heap",
			vacuumProgressLabels,
			nil,
		),
		phaseScanningHeap: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_scanning_heap",
			"VACUUM is currently scanning the heap",
			vacuumProgressLabels,
			nil,
		),
		phaseVacuumingIndexes: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_vacuuming_indexes",
			"VACUUM is currently vacuuming the indexes",
			vacuumProgressLabels,
			nil,
		),
		phaseVacuumingHeap: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_vacuuming_heap",
			"VACUUM is currently vacuuming the heap",
			vacuumProgressLabels,
			nil,
		),
		phaseCleaningUpIndexes: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_cleaning_up_indexes",
			"VACUUM is currently cleaning up indexes",
			vacuumProgressLabels,
			nil,
		),
		phaseTruncatingHeap: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_truncating_heap",
			"VACUUM is currently truncating the heap",
			vacuumProgressLabels,
			nil,
		),
		phasePerformingFinalCleanup: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_phase_performing_final_cleanup",
			"VACUUM is performing final cleanup",
			vacuumProgressLabels,
			nil,
		),
		heapBlksTotal: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_heap_blks_total",
			"Total number of heap blocks in the table",
			vacuumProgressLabels,
			nil,
		),
		heapBlksScanned: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_heap_blks_scanned",
			"Number of heap blocks scanned",
			vacuumProgressLabels,
			nil,
		),
		heapBlksVacuumed: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_heap_blks_vacuumed",
			"Number of heap blocks vacuumed",
			vacuumProgressLabels,
			nil,
		),
		indexVacuumCount: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_index_vacuum_count",
			"Number of completed index vacuum cycles",
			vacuumProgressLabels,
			nil,
		),
		maxDeadTuples: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_max_dead_tuples",
			"Number of dead tuples that we can store before needing to perform an index vacuum cycle",
			vacuumProgressLabels,
			nil,
		),
		numDeadTuples: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_num_dead_tuples",
			"Number of dead tuples collected since the last index vacuum cycle",
			vacuumProgressLabels,
			nil,
		),
		maxDeadTupleBytes: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_max_dead_tuple_bytes",
			"Amount of dead tuple data that we can store before needing to perform an index vacuum cycle (PostgreSQL 17+)",
			vacuumProgressLabels,
			nil,
		),
		deadTupleBytes: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_dead_tuple_bytes",
			"Amount of dead tuple data collected since the last index vacuum cycle (PostgreSQL 17+)",
			vacuumProgressLabels,
			nil,
		),
		numDeadItemIDs: prometheus.NewDesc(
			"postgres_stat_vacuum_progress_num_dead_item_ids",
			"Number of dead item identifiers collected since the last index vacuum cycle (PostgreSQL 17+)",
			vacuumProgressLabels,
			nil,
		),
	}
}

func (*statVacuumProgressScraper) Name() string {
	return "StatVacuumProgressScraper"
}

// phaseDesc maps a pg_stat_progress_vacuum phase to its metric. Unknown phases have no metric.
func (c *statVacuumProgressScraper) phaseDesc(phase string) (*prometheus.Desc, bool) {
	descs := map[string]*prometheus.Desc{
		"initializing":             c.phaseInitializing,
		"scanning heap":            c.phaseScanningHeap,
		"vacuuming indexes":        c.phaseVacuumingIndexes,
		"vacuuming heap":           c.phaseVacuumingHeap,
		"cleaning up indexes":      c.phaseCleaningUpIndexes,
		"truncating heap":          c.phaseTruncatingHeap,
		"performing final cleanup": c.phasePerformingFinalCleanup,
	}
	desc, ok := descs[phase]
	return desc, ok
}

func (c *statVacuumProgressScraper) Scrape(ctx context.Context, conn Querier, version Version, ch chan<- prometheus.Metric) error {
	query, ok := statVacuumProgressQueries.For(version)
	if !ok {
		return ErrUnsupportedVersion
	}
	bytesColumns := version.AtLeast(statVacuumProgressBytesVersion)

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var datname, schemaname, relname, phase string
	var heapBlksTotal, heapBlksScanned, heapBlksVacuumed, indexVacuumCount float64
	// Before PostgreSQL 17: max_dead_tuples, num_dead_tuples.
	// PostgreSQL 17+: max_dead_tuple_bytes, dead_tuple_bytes, num_dead_item_ids.
	var dead [3]float64

	for rows.Next() {
		dest := []any{
			&datname, &schemaname, &relname, &phase,
			&heapBlksTotal, &heapBlksScanned, &heapBlksVacuumed, &indexVacuumCount, &dead[0], &dead[1],
		}
		if bytesColumns {
			dest = append(dest, &dead[2])
		}
		if err := rows.Scan(dest...); err != nil {
			return err
		}

		labels := []string{datname, schemaname, relname}

		// postgres_stat_vacuum_progress_running
		ch <- prometheus.MustNewConstMetric(c.running, prometheus.GaugeValue, metricEnabled, labels...)
		if desc, ok := c.phaseDesc(phase); ok {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, metricEnabled, labels...)
		}
		ch <- prometheus.MustNewConstMetric(c.heapBlksTotal, prometheus.GaugeValue, heapBlksTotal, labels...)
		ch <- prometheus.MustNewConstMetric(c.heapBlksScanned, prometheus.GaugeValue, heapBlksScanned, labels...)
		ch <- prometheus.MustNewConstMetric(c.heapBlksVacuumed, prometheus.GaugeValue, heapBlksVacuumed, labels...)
		ch <- prometheus.MustNewConstMetric(c.indexVacuumCount, prometheus.GaugeValue, indexVacuumCount, labels...)

		if bytesColumns {
			ch <- prometheus.MustNewConstMetric(c.maxDeadTupleBytes, prometheus.GaugeValue, dead[0], labels...)
			ch <- prometheus.MustNewConstMetric(c.deadTupleBytes, prometheus.GaugeValue, dead[1], labels...)
			ch <- prometheus.MustNewConstMetric(c.numDeadItemIDs, prometheus.GaugeValue, dead[2], labels...)
		} else {
			ch <- prometheus.MustNewConstMetric(c.maxDeadTuples, prometheus.GaugeValue, dead[0], labels...)
			ch <- prometheus.MustNewConstMetric(c.numDeadTuples, prometheus.GaugeValue, dead[1], labels...)
		}
	}

	return rows.Err()
}
