package collector

import (
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// sample is one emitted metric, flattened for assertions.
type sample struct {
	labels map[string]string
	value  float64
}

// runScrape runs s against db and returns the emitted metrics keyed by metric name.
func runScrape(t *testing.T, s Scraper, db Querier, v Version) (map[string][]sample, error) {
	t.Helper()

	ch := make(chan prometheus.Metric, 256)
	err := s.Scrape(t.Context(), db, v, ch)
	close(ch)

	out := map[string][]sample{}
	for m := range ch {
		var pb dto.Metric
		require.NoError(t, m.Write(&pb))

		labels := map[string]string{}
		for _, lp := range pb.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		var value float64
		switch {
		case pb.GetCounter() != nil:
			value = pb.GetCounter().GetValue()
		case pb.GetGauge() != nil:
			value = pb.GetGauge().GetValue()
		default:
			t.Fatalf("unexpected metric type for %s", m.Desc())
		}
		name := descName(t, m.Desc())
		out[name] = append(out[name], sample{labels: labels, value: value})
	}
	return out, err
}

// descName extracts fqName from a Desc; client_golang does not export it.
func descName(t *testing.T, d *prometheus.Desc) string {
	t.Helper()

	s := d.String()
	const prefix = `Desc{fqName: "`
	require.Contains(t, s, prefix)
	s = s[len(prefix):]
	for i := range len(s) {
		if s[i] == '"' {
			return s[:i]
		}
	}
	t.Fatalf("cannot parse desc %s", d)
	return ""
}

// newMock returns a pgxmock connection that matches SQL exactly and fails on unexpected queries.
func newMock(t *testing.T) pgxmock.PgxConnIface {
	t.Helper()

	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
	})
	return mock
}

func mustVersion(t *testing.T, num string) Version {
	t.Helper()

	v, err := ParseVersion(num, num)
	require.NoError(t, err)
	return v
}
