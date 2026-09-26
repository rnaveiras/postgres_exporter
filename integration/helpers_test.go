package integration

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/rnaveiras/postgres_exporter/collector"
)

const (
	exporterRole     = "postgres_exporter"
	exporterPassword = "postgres_exporter"
)

// newDB clones the seeded template and returns the new database name. pgfresh drops it after the test.
func newDB(t *testing.T) string {
	t.Helper()

	cfg, err := pgx.ParseConfig(h.NewDB(t))
	require.NoError(t, err)
	return cfg.Database
}

// admin connects to dbname as the superuser.
func admin(t *testing.T, dbname string) *pgx.Conn {
	t.Helper()

	cfg, err := pgx.ParseConfig(adminDSN)
	require.NoError(t, err)
	cfg.Database = dbname
	conn, err := pgx.ConnectConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(t.Context())) })
	return conn
}

// exporterConfig is the connection the exporter uses: the monitoring role, pointed at dbname.
func exporterConfig(t *testing.T, dbname string) *pgx.ConnConfig {
	t.Helper()

	cfg, err := pgx.ParseConfig(adminDSN)
	require.NoError(t, err)
	cfg.User = exporterRole
	cfg.Password = exporterPassword
	cfg.Database = dbname
	cfg.RuntimeParams = map[string]string{"application_name": "postgres_exporter_test"}
	return cfg
}

func newRegistry(t *testing.T, dbname string, opts collector.Options) *prometheus.Registry {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	if testing.Verbose() {
		logger = slog.New(slog.NewTextHandler(testWriter{t}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewExporter(t.Context(), logger, exporterConfig(t, dbname), opts))
	return reg
}

// scrape runs one full exporter scrape as the monitoring role, with dbname as the main database.
func scrape(t *testing.T, dbname string, opts collector.Options) families {
	t.Helper()

	f, err := gather(newRegistry(t, dbname, opts))
	require.NoError(t, err)
	return f
}

// gather runs one scrape of reg. It does not fail the test, so it is safe inside an EventuallyWithT
// condition, which runs off the test goroutine.
func gather(reg prometheus.Gatherer) (families, error) {
	mfs, err := reg.Gather()
	if err != nil {
		return nil, err
	}
	out := families{}
	for _, mf := range mfs {
		out[mf.GetName()] = mf
	}
	return out, nil
}

type families map[string]*dto.MetricFamily

// values returns the value of every series of name whose labels include want.
func (f families) values(name string, want map[string]string) []float64 {
	var out []float64
	mf, ok := f[name]
	if !ok {
		return nil
	}
	for _, m := range mf.GetMetric() {
		if !matches(m, want) {
			continue
		}
		switch {
		case m.GetGauge() != nil:
			out = append(out, m.GetGauge().GetValue())
		case m.GetCounter() != nil:
			out = append(out, m.GetCounter().GetValue())
		case m.GetUntyped() != nil:
			out = append(out, m.GetUntyped().GetValue())
		default:
			// summaries and histograms have no single value
		}
	}
	return out
}

// value returns the single series of name matching want, failing the test when there is not exactly one.
func (f families) value(t *testing.T, name string, want map[string]string) float64 {
	t.Helper()

	v := f.values(name, want)
	require.Len(t, v, 1, "series %s%v", name, want)
	return v[0]
}

func (f families) has(name string) bool {
	_, ok := f[name]
	return ok
}

// labelValues returns the sorted distinct values of label across the series of name.
func (f families) labelValues(name, label string) []string {
	set := map[string]bool{}
	if mf, ok := f[name]; ok {
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == label {
					set[lp.GetValue()] = true
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(set))
}

func matches(m *dto.Metric, want map[string]string) bool {
	got := map[string]string{}
	for _, lp := range m.GetLabel() {
		got[lp.GetName()] = lp.GetValue()
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

// testWriter sends exporter logs to the test log in verbose runs.
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
