package collector

import (
	"log/slog"
	"testing"

	pgx "github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// a server that accepts the connection but cannot be queried is not up.
func TestCollector_Exporter_collectFrom_versionFailureIsDown(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	mock.ExpectQuery(versionQuery).WillReturnError(errMock)

	got := runCollect(t, newTestExporter(t), mock)
	require.Len(t, got["postgres_up"], 1)
	assert.InDelta(t, 0.0, got["postgres_up"][0].value, 0)
	assert.NotContains(t, got, "postgres_info")
}

// without the database list every per-database series vanishes, so the failure must be visible.
func TestCollector_Exporter_collectFrom_reportsListDatabases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want float64
	}{
		{name: "a failed list is reported as a failed scraper", err: errMock, want: 0},
		{name: "a successful list is reported as a successful scraper", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := newMock(t)
			expectVersionAndPlatform(mock)
			list := mock.ExpectQuery(listDatnameQuery).WithArgs([]string{})
			if tt.err != nil {
				list.WillReturnError(tt.err)
			} else {
				list.WillReturnRows(pgxmock.NewRows([]string{"datname"}))
			}

			got := runCollect(t, newTestExporter(t), mock)
			assert.InDelta(t, 1.0, got["postgres_up"][0].value, 0)
			success := find(got["postgres_exporter_scraper_success"], map[string]string{
				"scraper": listDatabasesScraperName, labelDatname: "postgres",
			})
			require.Len(t, success, 1)
			assert.InDelta(t, tt.want, success[0].value, 0)
		})
	}
}

// the exporter is rebuilt per request but connConfig is shared: a per-database scrape that changed it would
// point every later scrape at that database.
func TestCollector_Exporter_collectFrom_leavesConnConfigUnchanged(t *testing.T) {
	t.Parallel()

	mock := newMock(t)
	expectVersionAndPlatform(mock)
	mock.ExpectQuery(listDatnameQuery).WithArgs([]string{}).WillReturnRows(
		pgxmock.NewRows([]string{"datname"}).AddRow("app"))

	e := newTestExporter(t)
	got := runCollect(t, e, mock)
	assert.Equal(t, "postgres", e.connConfig.Database)
	// the per-database connection was attempted against its own copy, and failed on the closed port.
	connect := find(got["postgres_exporter_scraper_success"], map[string]string{
		"scraper": connectScraperName, labelDatname: "app",
	})
	require.Len(t, connect, 1)
	assert.InDelta(t, 0.0, connect[0].value, 0)
}

// newTestExporter returns an exporter without scrapers, so only collect's own queries run. Port 1 is closed,
// so a per-database connection fails fast.
func newTestExporter(t *testing.T) *Exporter {
	t.Helper()

	cfg, err := pgx.ParseConfig("host=127.0.0.1 port=1 dbname=postgres connect_timeout=1")
	require.NoError(t, err)
	return &Exporter{ctx: t.Context(), logger: slog.New(slog.DiscardHandler), connConfig: cfg}
}

func expectVersionAndPlatform(mock pgxmock.PgxConnIface) {
	mock.ExpectQuery(versionQuery).WillReturnRows(
		pgxmock.NewRows([]string{"num", "full"}).AddRow("170011", "17.11"))
	mock.ExpectQuery(platformQuery).WillReturnRows(
		pgxmock.NewRows([]string{"aurora", "rds", "cloudsql", "azure"}).AddRow(false, false, false, false))
}

// runCollect runs e.collectFrom against db and returns the emitted metrics keyed by metric name.
func runCollect(t *testing.T, e *Exporter, db Querier) map[string][]sample {
	t.Helper()

	ch := make(chan prometheus.Metric, 256)
	e.collectFrom(db, ch)
	close(ch)

	out := map[string][]sample{}
	for m := range ch {
		var pb dto.Metric
		require.NoError(t, m.Write(&pb))
		labels := map[string]string{}
		for _, lp := range pb.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		name := descName(t, m.Desc())
		out[name] = append(out[name], sample{labels: labels, value: pb.GetGauge().GetValue()})
	}
	return out
}

func find(samples []sample, want map[string]string) []sample {
	var out []sample
	for _, s := range samples {
		match := true
		for k, v := range want {
			if s.labels[k] != v {
				match = false
			}
		}
		if match {
			out = append(out, s)
		}
	}
	return out
}
