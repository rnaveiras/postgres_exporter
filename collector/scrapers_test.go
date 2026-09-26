package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errMock = errors.New("mock row error")

// pg_is_in_backup() does not exist on PostgreSQL 15+, so it must not be called there.
func TestInfoSkipsIsInBackupFrom15(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	mock := newMock(t)
	mock.ExpectQuery(isInRecoveryQuery).WillReturnRows(pgxmock.NewRows([]string{"r"}).AddRow(int64(0)))
	mock.ExpectQuery(startTimeQuery).WillReturnRows(pgxmock.NewRows([]string{"s"}).AddRow(now))
	mock.ExpectQuery(configLoadTimeQuery).WillReturnRows(pgxmock.NewRows([]string{"c"}).AddRow(now))

	got, err := runScrape(t, NewInfoScraper(), mock, mustVersion(t, "150019"))
	require.NoError(t, err)
	assert.NotContains(t, got, "postgres_is_in_backup")
	assert.Contains(t, got, "postgres_start_time_seconds")
}

func TestInfoReportsIsInBackupOn14(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	mock := newMock(t)
	mock.ExpectQuery(isInRecoveryQuery).WillReturnRows(pgxmock.NewRows([]string{"r"}).AddRow(int64(0)))
	mock.ExpectQuery(isInBackupQuery).WillReturnRows(pgxmock.NewRows([]string{"b"}).AddRow(int64(1)))
	mock.ExpectQuery(startTimeQuery).WillReturnRows(pgxmock.NewRows([]string{"s"}).AddRow(now))
	mock.ExpectQuery(configLoadTimeQuery).WillReturnRows(pgxmock.NewRows([]string{"c"}).AddRow(now))

	got, err := runScrape(t, NewInfoScraper(), mock, mustVersion(t, "140024"))
	require.NoError(t, err)
	require.Len(t, got["postgres_is_in_backup"], 1)
	assert.InDelta(t, 1.0, got["postgres_is_in_backup"][0].value, 0)
}

// On PostgreSQL 17+ pg_stat_bgwriter only has the background writer columns.
func TestStatBgwriterFrom17(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statBgwriter17).WillReturnRows(
		pgxmock.NewRows([]string{"buffers_clean", "maxwritten_clean", "buffers_alloc", "stats_reset"}).
			AddRow(int64(10), int64(2), int64(300), nil))

	got, err := runScrape(t, NewStatBgwriterScraper(), mock, mustVersion(t, "170011"))
	require.NoError(t, err)
	assert.Len(t, got, 3, "buffers_clean, maxwritten_clean and buffers_alloc; stats_reset is NULL")
	assert.Contains(t, got, "postgres_stat_bgwriter_buffers_alloc_total")
	assert.NotContains(t, got, "postgres_stat_bgwriter_checkpoints_timed_total")
}

func TestStatCheckpointerSkippedBefore17(t *testing.T) {
	t.Parallel()

	got, err := runScrape(t, NewStatCheckpointerScraper(), newMock(t), mustVersion(t, "160015"))
	require.ErrorIs(t, err, ErrUnsupportedVersion)
	assert.Empty(t, got)
}

func TestStatCheckpointer18(t *testing.T) {
	t.Parallel()

	reset := time.Unix(1_700_000_000, 0)
	mock := newMock(t)
	mock.ExpectQuery(statCheckpointer18).WillReturnRows(
		pgxmock.NewRows([]string{
			"num_timed", "num_requested", "restartpoints_timed", "restartpoints_req", "restartpoints_done",
			"write_time", "sync_time", "buffers_written", "stats_reset", "num_done", "slru_written",
		}).AddRow(int64(5), int64(1), int64(0), int64(0), int64(0), 2500.0, 500.0, int64(42), reset, int64(6), int64(7)))

	got, err := runScrape(t, NewStatCheckpointerScraper(), mock, mustVersion(t, "180006"))
	require.NoError(t, err)
	assert.Len(t, got, 11)
	assert.InDelta(t, 2.5, got["postgres_stat_checkpointer_write_time_seconds_total"][0].value, 1e-9)
	assert.InDelta(t, 6.0, got["postgres_stat_checkpointer_num_done_total"][0].value, 0)
}

// PostgreSQL 17 replaced max_dead_tuples/num_dead_tuples; the labels identify the table only.
func TestStatVacuumProgressFrom17(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statVacuumProgress17).WillReturnRows(
		pgxmock.NewRows([]string{
			"datname", "schemaname", "relname", "phase", "heap_blks_total", "heap_blks_scanned",
			"heap_blks_vacuumed", "index_vacuum_count", "max_dead_tuple_bytes", "dead_tuple_bytes", "num_dead_item_ids",
		}).AddRow("app", "public", "events", "scanning heap", 100.0, 50.0, 0.0, 0.0, 65536.0, 1024.0, 12.0))

	got, err := runScrape(t, NewStatVacuumProgressScraper(), mock, mustVersion(t, "170011"))
	require.NoError(t, err)

	require.Len(t, got["postgres_stat_vacuum_progress_phase_scanning_heap"], 1)
	assert.Equal(t,
		map[string]string{"datname": "app", "schemaname": "public", "relname": "events"},
		got["postgres_stat_vacuum_progress_phase_scanning_heap"][0].labels)
	assert.InDelta(t, 12.0, got["postgres_stat_vacuum_progress_num_dead_item_ids"][0].value, 0)
	assert.InDelta(t, 1024.0, got["postgres_stat_vacuum_progress_dead_tuple_bytes"][0].value, 0)
	assert.NotContains(t, got, "postgres_stat_vacuum_progress_num_dead_tuples")
}

var statDatabaseColumns = []string{
	"datname", "numbackends", "tup_returned", "tup_fetched", "tup_inserted", "tup_updated", "tup_deleted",
	"xact_commit", "xact_rollback", "blks_read", "blks_hit", "conflicts", "deadlocks", "temp_files", "temp_bytes",
}

// tup_deleted used to be emitted with the tup_updated value.
func TestStatDatabaseTupDeleted(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statDatabaseQuery).WillReturnRows(
		pgxmock.NewRows(statDatabaseColumns).
			AddRow("app", 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0, 13.0, 14.0))

	got, err := runScrape(t, NewStatDatabaseScraper(), mock, mustVersion(t, "180006"))
	require.NoError(t, err)
	assert.InDelta(t, 5.0, got["postgres_stat_database_tup_updated_total"][0].value, 0)
	assert.InDelta(t, 6.0, got["postgres_stat_database_tup_deleted_total"][0].value, 0)
}

// A row error must fail the scrape instead of being swallowed.
func TestStatDatabaseRowError(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statDatabaseQuery).WillReturnRows(
		pgxmock.NewRows(statDatabaseColumns).
			AddRow("app", 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0, 13.0, 14.0).
			AddRow("other", 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0, 13.0, 14.0).
			RowError(1, errMock))

	_, err := runScrape(t, NewStatDatabaseScraper(), mock, mustVersion(t, "180006"))
	require.ErrorIs(t, err, errMock)
}

// Replicas connected over a Unix socket have a NULL client_addr.
func TestStatReplicationNullClientAddr(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statReplicationLagBytes).WillReturnRows(
		pgxmock.NewRows([]string{"application_name", "client_addr", "state", "sync_state", "lag_bytes"}).
			AddRow("replica1", nil, "streaming", "async", 128.0).
			AddRow("replica2", "10.0.0.2", "streaming", "async", 0.0))

	got, err := runScrape(t, NewStatReplicationScraper(), mock, mustVersion(t, "180006"))
	require.NoError(t, err)
	require.Len(t, got["postgres_stat_replication_lag_bytes"], 2)
	assert.Empty(t, got["postgres_stat_replication_lag_bytes"][0].labels["client_addr"])
	assert.Equal(t, "10.0.0.2", got["postgres_stat_replication_lag_bytes"][1].labels["client_addr"])
}

// stats_reset is NULL on a cluster whose archiver statistics were never reset.
func TestStatArchiverNullStatsReset(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statArchiver).WillReturnRows(
		pgxmock.NewRows([]string{"archived_count", "failed_count", "stats_reset"}).AddRow(int64(3), int64(0), nil))

	got, err := runScrape(t, NewStatArchiverScraper(), mock, mustVersion(t, "180006"))
	require.NoError(t, err)
	assert.Contains(t, got, "postgres_stat_archiver_archived_total")
	assert.NotContains(t, got, "postgres_stat_archiver_stats_reset_timestamp")
}

func statUserTablesRows() *pgxmock.Rows {
	epoch := time.Unix(0, 0)
	return pgxmock.NewRows([]string{
		"schemaname", "relname", "seq_scan", "seq_tup_read", "idx_scan", "idx_tup_fetch", "n_tup_ins", "n_tup_upd",
		"n_tup_del", "n_tup_hot_upd", "n_live_tup", "n_dead_tup", "n_mod_since_analyze", "last_analyze",
		"last_autoanalyze", "last_vacuum", "last_autovacuum", "vacuum_count", "autovacuum_count", "analyze_count",
		"autoanalyze_count",
	}).AddRow("public", "events", 1.0, 2.0, 0.0, 0.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0,
		epoch, epoch, epoch, epoch, 0.0, 0.0, 0.0, 0.0)
}

func TestStatUserTablesHotUpdateNames(t *testing.T) {
	t.Parallel()

	for _, legacy := range []bool{true, false} {
		mock := newMock(t)
		mock.ExpectQuery(currentDatabaseQuery).WillReturnRows(pgxmock.NewRows([]string{"current_database"}).AddRow("app"))
		mock.ExpectQuery(statUserTablesQuery).WillReturnRows(statUserTablesRows())

		got, err := runScrape(t, NewStatUserTablesScraper(legacy), mock, mustVersion(t, "180006"))
		require.NoError(t, err)
		require.Len(t, got["postgres_stat_user_tables_n_tup_hot_upd_total"], 1)
		assert.InDelta(t, 6.0, got["postgres_stat_user_tables_n_tup_hot_upd_total"][0].value, 0)
		if legacy {
			assert.Contains(t, got, "postgres_stat_user_tables_n_tup_hot_upd")
		} else {
			assert.NotContains(t, got, "postgres_stat_user_tables_n_tup_hot_upd")
		}
	}
}

// servers older than the oldest supported major are scraped best-effort with the pre-17 view.
func TestCollector_statVacuumProgressScraper_Scrape_scrapesBelowMinSupported(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(statVacuumProgressPre17).WillReturnRows(
		pgxmock.NewRows([]string{
			"datname", "schemaname", "relname", "phase", "heap_blks_total", "heap_blks_scanned",
			"heap_blks_vacuumed", "index_vacuum_count", "max_dead_tuples", "num_dead_tuples",
		}).AddRow("app", "public", "events", "scanning heap", 100.0, 50.0, 0.0, 0.0, 1000.0, 12.0))

	got, err := runScrape(t, NewStatVacuumProgressScraper(), mock, mustVersion(t, "130022"))
	require.NoError(t, err)
	require.Len(t, got["postgres_stat_vacuum_progress_num_dead_tuples"], 1)
	assert.InDelta(t, 12.0, got["postgres_stat_vacuum_progress_num_dead_tuples"][0].value, 0)
}

// a relation dropped between the catalog read and the size call has a NULL size.
func TestCollector_diskUsageScraper_Scrape_skipsDroppedRelations(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(currentDatabaseQuery).WillReturnRows(pgxmock.NewRows([]string{"d"}).AddRow("app"))
	mock.ExpectQuery(tableUsageQuery).WillReturnRows(
		pgxmock.NewRows([]string{"schemaname", "tablename", "size"}).
			AddRow("pg_temp_3", "gone", nil).
			AddRow("public", "events", 8192.0))
	mock.ExpectQuery(indexUsageQuery).WillReturnRows(
		pgxmock.NewRows([]string{"schemaname", "tablename", "indexname", "size"}).
			AddRow("pg_temp_3", "gone", "gone_pkey", nil).
			AddRow("public", "events", "events_pkey", 16384.0))

	got, err := runScrape(t, NewDiskUsageScraper(), mock, mustVersion(t, "170011"))
	require.NoError(t, err)
	require.Len(t, got["postgres_disk_usage_table_bytes"], 1)
	assert.Equal(t, "events", got["postgres_disk_usage_table_bytes"][0].labels["tablename"])
	require.Len(t, got["postgres_disk_usage_index_bytes"], 1)
	assert.Equal(t, "events_pkey", got["postgres_disk_usage_index_bytes"][0].labels["indexname"])
}
