package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rnaveiras/postgres_exporter/collector"
)

var defaultOpts = collector.Options{LegacyNames: true}

// Every scraper succeeds on every supported major, connected as the monitoring role.
func TestAllScrapersSucceed(t *testing.T) {
	db := newDB(t)
	f := scrape(t, db, defaultOpts)

	assert.InDelta(t, 1.0, f.value(t, "postgres_up", nil), 0)

	mf, ok := f["postgres_exporter_scraper_success"]
	require.True(t, ok)
	scrapers := map[string]bool{}
	for _, m := range mf.GetMetric() {
		labels := map[string]string{}
		for _, lp := range m.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		scrapers[labels["scraper"]] = true
		assert.InDelta(t, 1.0, m.GetGauge().GetValue(), 0, "scraper %s on %s", labels["scraper"], labels["datname"])
	}
	for _, s := range []string{
		"InfoScraper", "LocksScraper", "StatActivityScraper", "StatArchiverScraper", "StatBgwriterScraper",
		"StatCheckpointerScraper", "StatDatabaseScraper", "StatReplicationScraper", "connect", "list_databases",
		"StatVacuumProgressScraper", "StatUserTablesScraper", "StatUserIndexesScraper", "DiskUsageScraper",
	} {
		assert.True(t, scrapers[s], "scraper %s reported", s)
	}
}

// postgres_info carries the server's own version strings and the detected platform.
func TestInfo(t *testing.T) {
	db := newDB(t)
	f := scrape(t, db, defaultOpts)

	assert.InDelta(t, 1.0, f.value(t, "postgres_info", map[string]string{
		"version":     strings.Fields(pgFull)[0],
		"version_num": pgNum,
		"platform":    string(collector.PlatformCommunity),
	}), 0)
	// the experimental CI job runs a major newer than the newest tested one.
	unsupported, untested := 0.0, 0.0
	if pgMajor < collector.MinSupportedVersion {
		unsupported = 1
	}
	if pgMajor > collector.MaxTestedVersion {
		untested = 1
	}
	assert.InDelta(t, unsupported, f.value(t, "postgres_exporter_unsupported_version", nil), 0)
	assert.InDelta(t, untested, f.value(t, "postgres_exporter_untested_version", nil), 0)
}

// Metrics that exist only on some majors appear exactly there.
func TestVersionGatedMetrics(t *testing.T) {
	db := newDB(t)
	f := scrape(t, db, defaultOpts)

	assert.Equal(t, pgMajor < 15, f.has("postgres_is_in_backup"), "postgres_is_in_backup")
	assert.True(t, f.has("postgres_start_time_seconds"))

	assert.Equal(t, pgMajor < 17, f.has("postgres_stat_bgwriter_checkpoints_timed_total"), "bgwriter checkpoint columns")
	assert.True(t, f.has("postgres_stat_bgwriter_buffers_alloc_total"))

	assert.Equal(t, pgMajor >= 17, f.has("postgres_stat_checkpointer_num_timed_total"), "checkpointer")
	assert.Equal(t, pgMajor >= 18, f.has("postgres_stat_checkpointer_num_done_total"), "checkpointer num_done")
}

// The descriptors pass Prometheus' metric lint.
func TestLint(t *testing.T) {
	db := newDB(t)
	problems, err := testutil.GatherAndLint(newRegistry(t, db, collector.Options{LegacyNames: false}))
	require.NoError(t, err)
	assert.Empty(t, problems)
}

// tup_deleted reports deleted rows, not updated ones.
func TestStatDatabaseTupDeleted(t *testing.T) {
	db := newDB(t)
	conn := admin(t, db)
	ctx := t.Context()

	_, err := conn.Exec(ctx, `INSERT INTO app_events (payload) SELECT 'x' FROM generate_series(1, 100)`)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `DELETE FROM app_events WHERE id <= 40`)
	require.NoError(t, err)
	flushStats(t, conn)

	want := map[string]string{"datname": db}
	reg := newRegistry(t, db, defaultOpts)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		// PG14 sends a session's pending statistics at most every 500ms, and only when it finishes a
		// statement, so keep the writing session busy until they arrive.
		_, err := conn.Exec(ctx, `SELECT 1`)
		require.NoError(c, err)
		f, err := gather(reg)
		require.NoError(c, err)
		v := f.values("postgres_stat_database_tup_deleted_total", want)
		require.Len(c, v, 1)
		assert.GreaterOrEqual(c, v[0], 40.0)
	}, 10*time.Second, 200*time.Millisecond)

	f := scrape(t, db, defaultOpts)
	assert.Less(t, f.value(t, "postgres_stat_database_tup_updated_total", want), 40.0)
}

// Table sizes are looked up by OID, so quoted and dotted names work.
func TestDiskUsageQuotedNames(t *testing.T) {
	db := newDB(t)
	f := scrape(t, db, defaultOpts)

	assert.InDelta(t, 1.0, f.value(t, "postgres_exporter_scraper_success", map[string]string{"scraper": "DiskUsageScraper", "datname": db}), 0)
	f.value(t, "postgres_disk_usage_table_bytes", map[string]string{"datname": db, "schemaname": "Mixed Case", "tablename": "Weird.Name"})
}

// Tables without indexes are reported.
func TestTableWithoutIndexes(t *testing.T) {
	db := newDB(t)
	f := scrape(t, db, defaultOpts)

	want := map[string]string{"datname": db, "schemaname": "public", "relname": "app_no_index"}
	assert.InDelta(t, 0.0, f.value(t, "postgres_stat_user_tables_idx_scan_total", want), 0)
	f.value(t, "postgres_stat_user_tables_seq_scan_total", want)
}

// The deprecated name is emitted only with legacy names enabled.
func TestHotUpdateNames(t *testing.T) {
	db := newDB(t)
	want := map[string]string{"datname": db, "relname": "app_events"}

	f := scrape(t, db, collector.Options{LegacyNames: true})
	f.value(t, "postgres_stat_user_tables_n_tup_hot_upd_total", want)
	f.value(t, "postgres_stat_user_tables_n_tup_hot_upd", want)
	assert.Contains(t, f["postgres_stat_user_tables_n_tup_hot_upd"].GetHelp(), "(DEPRECATED)")

	f = scrape(t, db, collector.Options{LegacyNames: false})
	f.value(t, "postgres_stat_user_tables_n_tup_hot_upd_total", want)
	assert.False(t, f.has("postgres_stat_user_tables_n_tup_hot_upd"))
}

// A running VACUUM shows up with the columns of the running major.
func TestVacuumProgress(t *testing.T) {
	db := newDB(t)
	conn := admin(t, db)
	ctx := t.Context()

	_, err := conn.Exec(ctx, `INSERT INTO app_events (payload) SELECT repeat('x', 200) FROM generate_series(1, 20000)`)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `DELETE FROM app_events WHERE id % 2 = 0`)
	require.NoError(t, err)

	// A cost limit of 1 with a delay makes VACUUM sleep after nearly every page, so it stays visible.
	vacuum := admin(t, db)
	var pid int
	require.NoError(t, vacuum.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid))
	_, err = vacuum.Exec(ctx, `SET vacuum_cost_delay = '20ms'`)
	require.NoError(t, err)
	_, err = vacuum.Exec(ctx, `SET vacuum_cost_limit = 1`)
	require.NoError(t, err)

	canceller := admin(t, db) // opened now: t.Context() is already canceled when cleanups run
	done := make(chan error, 1)
	go func() {
		_, err := vacuum.Exec(context.WithoutCancel(ctx), `VACUUM app_events`)
		done <- err
	}()
	t.Cleanup(func() {
		if _, err := canceller.Exec(context.WithoutCancel(t.Context()), `SELECT pg_cancel_backend($1)`, pid); err != nil {
			t.Logf("cancel vacuum: %v", err)
		}
		<-done
	})

	want := map[string]string{"datname": db, "schemaname": "public", "relname": "app_events"}
	var f families
	reg := newRegistry(t, db, defaultOpts)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var err error
		f, err = gather(reg)
		require.NoError(c, err)
		assert.Len(c, f.values("postgres_stat_vacuum_progress_running", want), 1)
	}, 10*time.Second, 50*time.Millisecond)

	if pgMajor >= 17 {
		f.value(t, "postgres_stat_vacuum_progress_num_dead_item_ids", want)
		f.value(t, "postgres_stat_vacuum_progress_max_dead_tuple_bytes", want)
		assert.False(t, f.has("postgres_stat_vacuum_progress_num_dead_tuples"))
	} else {
		f.value(t, "postgres_stat_vacuum_progress_num_dead_tuples", want)
		assert.False(t, f.has("postgres_stat_vacuum_progress_num_dead_item_ids"))
	}
	for _, label := range []string{"pid", "query_start"} {
		assert.Empty(t, f.labelValues("postgres_stat_vacuum_progress_running", label), "label %s", label)
	}
}

// Dropping a scraped database between scrapes does not affect postgres_up or the global scrapers.
func TestDroppedDatabase(t *testing.T) {
	main := newDB(t)
	gone := newDB(t)

	// one registry for both scrapes, as the HTTP handler shares one connection config across requests: a
	// per-database scrape that changed it would point the second scrape at another database.
	reg := newRegistry(t, main, defaultOpts)
	f, err := gather(reg)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, f.value(t, "postgres_exporter_scraper_success", map[string]string{"scraper": "connect", "datname": gone}), 0)

	_, err = admin(t, main).Exec(t.Context(), `DROP DATABASE `+pgx.Identifier{gone}.Sanitize()+` WITH (FORCE)`)
	require.NoError(t, err)

	f, err = gather(reg)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, f.value(t, "postgres_up", nil), 0)
	assert.InDelta(t, 1.0, f.value(t, "postgres_exporter_scraper_success", map[string]string{"scraper": "InfoScraper", "datname": main}), 0)
	assert.Empty(t, f.values("postgres_exporter_scraper_success", map[string]string{"datname": gone}))
}

// A database the monitoring role cannot connect to fails only its own scrapers.
func TestUnreachableDatabase(t *testing.T) {
	main := newDB(t)
	blocked := newDB(t)
	other := newDB(t)

	_, err := admin(t, main).Exec(t.Context(), `ALTER DATABASE `+pgx.Identifier{blocked}.Sanitize()+` CONNECTION LIMIT 0`)
	require.NoError(t, err)

	f := scrape(t, main, defaultOpts)
	assert.InDelta(t, 1.0, f.value(t, "postgres_up", nil), 0)
	assert.InDelta(t, 0.0, f.value(t, "postgres_exporter_scraper_success", map[string]string{"scraper": "connect", "datname": blocked}), 0)
	assert.InDelta(t, 1.0, f.value(t, "postgres_exporter_scraper_success", map[string]string{"scraper": "connect", "datname": other}), 0)
	assert.InDelta(t, 1.0, f.value(t, "postgres_exporter_scraper_success", map[string]string{"scraper": "StatUserTablesScraper", "datname": other}), 0)
}

// Every query variant for the running major parses and describes as the monitoring role.
func TestAllQueriesPrepare(t *testing.T) {
	db := newDB(t)
	conn, err := pgx.ConnectConfig(t.Context(), exporterConfig(t, db))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(t.Context())) })

	v, err := collector.ParseVersion(pgNum, pgFull)
	require.NoError(t, err)

	prepared := 0
	for _, q := range collector.AllQueries() {
		t.Run(q.Name, func(t *testing.T) {
			if !q.AppliesTo(v) {
				t.Skipf("applies to PostgreSQL %d..%d", q.MinMajor, q.MaxMajor)
			}
			_, err := conn.Prepare(t.Context(), q.Name, q.SQL)
			require.NoError(t, err)
			prepared++
		})
	}
	require.Positive(t, prepared)
}

// flushStats makes this session's statistics visible now on PG15+; PG14 flushes on its own schedule.
func flushStats(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	if pgMajor >= 15 {
		_, err := conn.Exec(t.Context(), `SELECT pg_stat_force_next_flush()`)
		require.NoError(t, err)
		_, err = conn.Exec(t.Context(), `SELECT 1`)
		require.NoError(t, err)
	}
}
