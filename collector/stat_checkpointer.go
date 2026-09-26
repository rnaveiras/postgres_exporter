package collector

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus"
)

// pg_stat_checkpointer exists since PostgreSQL 17, when the checkpoint columns moved out of pg_stat_bgwriter.
// PostgreSQL 18 adds num_done and slru_written.
const (
	statCheckpointerVersion        = 17
	statCheckpointerNumDoneVersion = 18

	statCheckpointer17 = `
SELECT num_timed
     , num_requested
     , restartpoints_timed
     , restartpoints_req
     , restartpoints_done
     , write_time
     , sync_time
     , buffers_written
     , stats_reset
  FROM pg_stat_checkpointer /*postgres_exporter*/`

	statCheckpointer18 = `
SELECT num_timed
     , num_requested
     , restartpoints_timed
     , restartpoints_req
     , restartpoints_done
     , write_time
     , sync_time
     , buffers_written
     , stats_reset
     , num_done
     , slru_written
  FROM pg_stat_checkpointer /*postgres_exporter*/`
)

var statCheckpointerQueries = versioned{
	statCheckpointerVersion:        statCheckpointer17,
	statCheckpointerNumDoneVersion: statCheckpointer18,
}

type statCheckpointerScraper struct {
	numTimed           *prometheus.Desc
	numRequested       *prometheus.Desc
	restartpointsTimed *prometheus.Desc
	restartpointsReq   *prometheus.Desc
	restartpointsDone  *prometheus.Desc
	writeTime          *prometheus.Desc
	syncTime           *prometheus.Desc
	buffersWritten     *prometheus.Desc
	statsReset         *prometheus.Desc
	numDone            *prometheus.Desc
	slruWritten        *prometheus.Desc
}

// NewStatCheckpointerScraper returns a new Scraper exposing the PostgreSQL 17+ `pg_stat_checkpointer` view.
func NewStatCheckpointerScraper() Scraper {
	return &statCheckpointerScraper{
		numTimed: prometheus.NewDesc(
			"postgres_stat_checkpointer_num_timed_total",
			"Number of scheduled checkpoints due to timeout",
			nil,
			nil,
		),
		numRequested: prometheus.NewDesc(
			"postgres_stat_checkpointer_num_requested_total",
			"Number of requested checkpoints",
			nil,
			nil,
		),
		restartpointsTimed: prometheus.NewDesc(
			"postgres_stat_checkpointer_restartpoints_timed_total",
			"Number of scheduled restartpoints due to timeout or after a failed attempt to perform it",
			nil,
			nil,
		),
		restartpointsReq: prometheus.NewDesc(
			"postgres_stat_checkpointer_restartpoints_req_total",
			"Number of requested restartpoints",
			nil,
			nil,
		),
		restartpointsDone: prometheus.NewDesc(
			"postgres_stat_checkpointer_restartpoints_done_total",
			"Number of restartpoints that have been performed",
			nil,
			nil,
		),
		writeTime: prometheus.NewDesc(
			"postgres_stat_checkpointer_write_time_seconds_total",
			"Total time spent in the portion of processing checkpoints and restartpoints where files are written to disk",
			nil,
			nil,
		),
		syncTime: prometheus.NewDesc(
			"postgres_stat_checkpointer_sync_time_seconds_total",
			"Total time spent in the portion of processing checkpoints and restartpoints where files are synchronized to disk",
			nil,
			nil,
		),
		buffersWritten: prometheus.NewDesc(
			"postgres_stat_checkpointer_buffers_written_total",
			"Number of shared buffers written during checkpoints and restartpoints",
			nil,
			nil,
		),
		statsReset: prometheus.NewDesc(
			"postgres_stat_checkpointer_stats_reset_timestamp_seconds",
			"Time at which these statistics were last reset",
			nil,
			nil,
		),
		numDone: prometheus.NewDesc(
			"postgres_stat_checkpointer_num_done_total",
			"Number of checkpoints that have been performed (PostgreSQL 18+)",
			nil,
			nil,
		),
		slruWritten: prometheus.NewDesc(
			"postgres_stat_checkpointer_slru_written_total",
			"Number of SLRU buffers written during checkpoints and restartpoints (PostgreSQL 18+)",
			nil,
			nil,
		),
	}
}

func (*statCheckpointerScraper) Name() string {
	return "StatCheckpointerScraper"
}

type statCheckpointerRow struct {
	numTimed, numRequested, restartpointsTimed, restartpointsReq, restartpointsDone, buffersWritten int64
	writeTime, syncTime                                                                             float64
	statsReset                                                                                      pgtype.Timestamptz
	numDone, slruWritten                                                                            int64
}

func (c *statCheckpointerScraper) Scrape(ctx context.Context, conn Querier, version Version, ch chan<- prometheus.Metric) error {
	query, ok := statCheckpointerQueries.For(version)
	if !ok {
		return ErrUnsupportedVersion
	}
	withNumDone := version.AtLeast(statCheckpointerNumDoneVersion)

	var r statCheckpointerRow
	dest := []any{
		&r.numTimed, &r.numRequested, &r.restartpointsTimed, &r.restartpointsReq, &r.restartpointsDone,
		&r.writeTime, &r.syncTime, &r.buffersWritten, &r.statsReset,
	}
	if withNumDone {
		dest = append(dest, &r.numDone, &r.slruWritten)
	}
	if err := conn.QueryRow(ctx, query).Scan(dest...); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(c.numTimed, prometheus.CounterValue, float64(r.numTimed))
	ch <- prometheus.MustNewConstMetric(c.numRequested, prometheus.CounterValue, float64(r.numRequested))
	ch <- prometheus.MustNewConstMetric(c.restartpointsTimed, prometheus.CounterValue, float64(r.restartpointsTimed))
	ch <- prometheus.MustNewConstMetric(c.restartpointsReq, prometheus.CounterValue, float64(r.restartpointsReq))
	ch <- prometheus.MustNewConstMetric(c.restartpointsDone, prometheus.CounterValue, float64(r.restartpointsDone))
	ch <- prometheus.MustNewConstMetric(c.writeTime, prometheus.CounterValue, r.writeTime/millisecondsPerSecond)
	ch <- prometheus.MustNewConstMetric(c.syncTime, prometheus.CounterValue, r.syncTime/millisecondsPerSecond)
	ch <- prometheus.MustNewConstMetric(c.buffersWritten, prometheus.CounterValue, float64(r.buffersWritten))
	if r.statsReset.Valid {
		ch <- prometheus.MustNewConstMetric(c.statsReset, prometheus.GaugeValue, float64(r.statsReset.Time.Unix()))
	}
	if withNumDone {
		ch <- prometheus.MustNewConstMetric(c.numDone, prometheus.CounterValue, float64(r.numDone))
		ch <- prometheus.MustNewConstMetric(c.slruWritten, prometheus.CounterValue, float64(r.slruWritten))
	}
	return nil
}
