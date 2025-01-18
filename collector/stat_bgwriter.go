package collector

import (
	"context"
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	statBgwriter = `
SELECT checkpoints_timed
     , checkpoints_req
     , checkpoint_write_time
     , checkpoint_sync_time
     , buffers_checkpoint
     , buffers_clean
     , maxwritten_clean
     , buffers_backend
     , buffers_backend_fsync
     , buffers_alloc
     , stats_reset
  FROM pg_stat_bgwriter /*postgres_exporter*/`
)

type statBgwriterScraper struct {
	checkpointsTimed    *prometheus.Desc
	checkpointsReq      *prometheus.Desc
	checkpointWriteTime *prometheus.Desc
	checkpointSyncTime  *prometheus.Desc
	buffersCheckpoint   *prometheus.Desc
	buffersClean        *prometheus.Desc
	maxWrittenClean     *prometheus.Desc
	buffersBackend      *prometheus.Desc
	buffersBackendFsync *prometheus.Desc
	buffersAlloc        *prometheus.Desc
	statsReset          *prometheus.Desc
}

// NewStatBgwriterScraper returns a new Scraper exposing PostgreSQL `pg_stat_bgwriter` view
func NewStatBgwriterScraper() Scraper {
	return &statBgwriterScraper{
		checkpointsTimed: prometheus.NewDesc(
			"postgres_stat_bgwriter_checkpoints_timed_total",
			"Number of scheduled checkpoints that have been performed",
			nil,
			nil,
		),
		checkpointsReq: prometheus.NewDesc(
			"postgres_stat_bgwriter_checkpoints_req_total",
			"Number of requested checkpoints that have been performed",
			nil,
			nil,
		),
		checkpointWriteTime: prometheus.NewDesc(
			"postgres_stat_bgwriter_checkpoint_write_time_seconds_total",
			"Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk",
			nil,
			nil,
		),
		checkpointSyncTime: prometheus.NewDesc(
			"postgres_stat_bgwriter_checkpoint_sync_time_seconds_total",
			"Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk",
			nil,
			nil,
		),
		buffersCheckpoint: prometheus.NewDesc(
			"postgres_stat_bgwriter_buffers_checkpoint_total",
			"Number of buffers written during checkpoints",
			nil,
			nil,
		),
		buffersClean: prometheus.NewDesc(
			"postgres_stat_bgwriter_buffers_clean_total",
			"Number of buffers written by the background writer",
			nil,
			nil,
		),
		maxWrittenClean: prometheus.NewDesc(
			"postgres_stat_bgwriter_maxwritten_clean_total",
			"Number of times the background writer stopped a cleaning scan because it had written too many buffers",
			nil,
			nil,
		),
		buffersBackend: prometheus.NewDesc(
			"postgres_stat_bgwriter_buffers_backend_total",
			"Number of buffers written directly by a backend",
			nil,
			nil,
		),
		buffersBackendFsync: prometheus.NewDesc(
			"postgres_stat_bgwriter_buffers_backend_fsync_total",
			"Number of times a backend had to execute its own fsync call",
			nil,
			nil,
		),
		buffersAlloc: prometheus.NewDesc(
			"postgres_stat_bgwriter_buffers_alloc_total",
			"Number of buffers allocated",
			nil,
			nil,
		),
		statsReset: prometheus.NewDesc(
			"postgres_stat_bgwriter_stats_reset_timestamp",
			"Time at which these statistics were last reset",
			nil,
			nil,
		),
	}
}

func (*statBgwriterScraper) Name() string {
	return "StatBgwriterScraper"
}

func (c *statBgwriterScraper) Scrape(ctx context.Context, conn *pgx.Conn, _ Version, ch chan<- prometheus.Metric) error {
	var checkpointsTimedCounter, checkpointsReqCounter,
		buffersCheckpoint, buffersClean, maxWrittenClean,
		buffersBackend, buffersBackendFsync, buffersAlloc int64
	var checkpointWriteTime, checkpointSyncTime float64
	var statsReset time.Time

	if err := conn.QueryRow(ctx, statBgwriter).
		Scan(&checkpointsTimedCounter,
			&checkpointsReqCounter,
			&checkpointWriteTime,
			&checkpointSyncTime,
			&buffersCheckpoint,
			&buffersClean,
			&maxWrittenClean,
			&buffersBackend,
			&buffersBackendFsync,
			&buffersAlloc,
			&statsReset,
		); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(c.checkpointsTimed, prometheus.CounterValue, float64(checkpointsTimedCounter))
	ch <- prometheus.MustNewConstMetric(c.checkpointsReq, prometheus.CounterValue, float64(checkpointsReqCounter))
	ch <- prometheus.MustNewConstMetric(c.checkpointWriteTime, prometheus.CounterValue, float64(checkpointWriteTime/1000))
	ch <- prometheus.MustNewConstMetric(c.checkpointSyncTime, prometheus.CounterValue, float64(checkpointSyncTime/1000))
	ch <- prometheus.MustNewConstMetric(c.buffersCheckpoint, prometheus.CounterValue, float64(buffersCheckpoint))
	ch <- prometheus.MustNewConstMetric(c.buffersClean, prometheus.CounterValue, float64(buffersClean))
	ch <- prometheus.MustNewConstMetric(c.maxWrittenClean, prometheus.CounterValue, float64(maxWrittenClean))
	ch <- prometheus.MustNewConstMetric(c.buffersBackend, prometheus.CounterValue, float64(buffersBackend))
	ch <- prometheus.MustNewConstMetric(c.buffersBackendFsync, prometheus.CounterValue, float64(buffersBackendFsync))
	ch <- prometheus.MustNewConstMetric(c.buffersAlloc, prometheus.CounterValue, float64(buffersAlloc))
	ch <- prometheus.MustNewConstMetric(c.statsReset, prometheus.GaugeValue, float64(statsReset.UTC().Unix()))
	return nil
}
