package collector

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		num, full string
		major     int
		minor     int
		str       string
	}{
		{num: "140024", full: "14.24", major: 14, minor: 24, str: "14.24"},
		{num: "160010", full: "16.10", major: 16, minor: 10, str: "16.10"},
		{num: "170011", full: "17.11", major: 17, minor: 11, str: "17.11"},
		{num: "190000", full: "19beta4", major: 19, minor: 0, str: "19.0"},
	}
	for _, tt := range tests {
		t.Run(tt.num, func(t *testing.T) {
			t.Parallel()

			v, err := ParseVersion(tt.num, tt.full)
			require.NoError(t, err)
			assert.Equal(t, tt.major, v.Major())
			assert.Equal(t, tt.minor, v.Minor())
			assert.Equal(t, tt.str, v.String())
			assert.Equal(t, tt.full, v.Full)
		})
	}
}

// packaged builds append a suffix to server_version that changes on every package rebuild.
func TestCollector_ParseVersion_stripsPackagerSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, num, full, want string
	}{
		{name: "debian pgdg suffix is dropped", num: "170006", full: "17.6 (Debian 17.6-1.pgdg12+1)", want: "17.6"},
		{name: "ubuntu suffix is dropped", num: "160004", full: "16.4 (Ubuntu 16.4-1.pgdg22.04+1)", want: "16.4"},
		{name: "plain release is kept", num: "170011", full: "17.11", want: "17.11"},
		{name: "beta is kept", num: "190000", full: "19beta4", want: "19beta4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := ParseVersion(tt.num, tt.full)
			require.NoError(t, err)
			assert.Equal(t, tt.want, v.Full)
		})
	}
}

func TestParseVersionErrors(t *testing.T) {
	t.Parallel()

	for _, num := range []string{"", "17.11", "17beta1", "904"} {
		_, err := ParseVersion(num, num)
		require.Error(t, err, num)
	}
}

func TestVersionComparisons(t *testing.T) {
	t.Parallel()

	v := mustVersion(t, "170011")
	assert.True(t, v.AtLeast(17))
	assert.False(t, v.AtLeast(18))
	assert.True(t, v.Before(18))
	assert.False(t, v.Before(17))
}

func TestVersionedFor(t *testing.T) {
	t.Parallel()

	q := versioned{14: "pre17", 17: "17", 18: "18"}
	tests := map[string]struct {
		want string
		ok   bool
	}{
		"130022": {want: "", ok: false},
		"140024": {want: "pre17", ok: true},
		"160015": {want: "pre17", ok: true},
		"170011": {want: "17", ok: true},
		"180006": {want: "18", ok: true},
		"190000": {want: "18", ok: true},
	}
	for num, tt := range tests {
		got, ok := q.For(mustVersion(t, num))
		assert.Equal(t, tt.ok, ok, num)
		assert.Equal(t, tt.want, got, num)
	}
}

// Every variant table must give exactly one query for each major from the floor to the newest tested + 1.
func TestAllQueriesVariantsCoverEveryMajor(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"stat_bgwriter", "stat_checkpointer", "stat_progress_vacuum"} {
		for major := MinSupportedVersion; major <= MaxTestedVersion+1; major++ {
			v := Version{Num: major * versionNumDivisor}
			n := 0
			for _, q := range AllQueries() {
				if strings.HasPrefix(q.Name, name+"_pg") && q.AppliesTo(v) {
					n++
				}
			}
			if name == "stat_checkpointer" && major < statCheckpointerVersion {
				assert.Zero(t, n, "%s on %d", name, major)
				continue
			}
			assert.Equal(t, 1, n, "%s on %d", name, major)
		}
	}
}

func TestVersionLoggerDue(t *testing.T) {
	t.Parallel()

	l := &versionLogger{last: map[int]time.Time{}}
	assert.True(t, l.due(13, time.Hour))
	assert.False(t, l.due(13, time.Hour))
	assert.True(t, l.due(19, 0))
	assert.False(t, l.due(19, 0))
}
