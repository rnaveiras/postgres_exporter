package integration

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rnaveiras/postgres_exporter/collector"
)

var update = flag.Bool("update", false, "rewrite testdata/metrics_pg<major>.golden from the running server")

// keptLabelValues are labels whose values the exporter controls, so they belong in the golden file.
var keptLabelValues = map[string]bool{"scraper": true, "state": true}

// TestGolden compares the shape of a full scrape (metric names, types and label names) with the golden
// file for the running major. The diff between two golden files is the metric difference between majors.
func TestGolden(t *testing.T) {
	db := newDB(t)
	mfs, err := newRegistry(t, db, defaultOpts).Gather()
	require.NoError(t, err)

	got := normalize(mfs)
	path := filepath.Join("testdata", "metrics_pg"+strconv.Itoa(pgMajor)+".golden")
	if *update {
		require.NoError(t, os.WriteFile(path, []byte(got), 0o600))
		return
	}

	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) && pgMajor > collector.MaxTestedVersion {
		// an untested major (the experimental CI job) has no golden file until it becomes supported.
		t.Skipf("no golden file for untested PostgreSQL %d", pgMajor)
	}
	require.NoError(t, err, "run `go test -run TestGolden -update` to create it")
	assert.Equal(t, string(want), got)
}

// normalize renders one line per distinct series shape, sorted. Values are dropped, and so are label
// values except the ones in keptLabelValues. Stats reset timestamps depend on the cluster's history, not
// on its version, so they are left out.
func normalize(mfs []*dto.MetricFamily) string {
	set := map[string]bool{}
	for _, mf := range mfs {
		name := mf.GetName()
		if strings.Contains(name, "stats_reset") {
			continue
		}
		set["# TYPE "+name+" "+strings.ToLower(mf.GetType().String())] = true
		for _, m := range mf.GetMetric() {
			labels := make([]string, 0, len(m.GetLabel()))
			for _, lp := range m.GetLabel() {
				l := lp.GetName()
				if keptLabelValues[l] {
					l += "=" + strconv.Quote(lp.GetValue())
				}
				labels = append(labels, l)
			}
			slices.Sort(labels)
			set[name+"{"+strings.Join(labels, ",")+"}"] = true
		}
	}

	lines := make([]string, 0, len(set))
	for l := range set {
		lines = append(lines, l)
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n") + "\n"
}
