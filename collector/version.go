package collector

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

const (
	// MinSupportedVersion is the oldest Postgres major the exporter is tested against. Older servers are
	// scraped on a best-effort basis and reported by postgres_exporter_unsupported_version.
	MinSupportedVersion = 14
	// MaxTestedVersion is the newest Postgres major the exporter is tested against. Newer servers use the
	// newest query variants and are reported by postgres_exporter_untested_version.
	MaxTestedVersion = 18

	versionNumDivisor = 10000

	// bestEffortVersion keys the oldest variant of a versioned query, so servers older than
	// MinSupportedVersion still run it instead of reporting no metrics.
	bestEffortVersion = 0
)

// ErrUnsupportedVersion is returned by a scraper whose views do not exist on the running Postgres major.
// The exporter reports such a scraper as successful with no metrics.
var ErrUnsupportedVersion = errors.New("not available on this Postgres version")

// Version identifies the Postgres server being scraped.
type Version struct {
	// Num is server_version_num, e.g. 170011 for 17.11 and 190000 for 19beta4.
	Num int
	// Full is server_version without the packager suffix, e.g. "17.11" or "19beta4" (Debian reports
	// "17.6 (Debian 17.6-1.pgdg12+1)", whose suffix changes on every package rebuild).
	Full string
}

// ParseVersion builds a Version from the server_version_num and server_version settings.
func ParseVersion(num, full string) (Version, error) {
	n, err := strconv.Atoi(num)
	if err != nil {
		return Version{}, fmt.Errorf("parse server_version_num %q: %w", num, err)
	}
	if n < versionNumDivisor {
		return Version{}, fmt.Errorf("parse server_version_num %q: value too small", num)
	}
	release, _, _ := strings.Cut(full, " ")
	return Version{Num: n, Full: release}, nil
}

// Major returns the major version, e.g. 17.
func (v Version) Major() int { return v.Num / versionNumDivisor }

// Minor returns the minor version, e.g. 11 for 17.11.
func (v Version) Minor() int { return v.Num % versionNumDivisor }

// AtLeast reports whether the server major is at least major.
func (v Version) AtLeast(major int) bool { return v.Major() >= major }

// Before reports whether the server major is older than major.
func (v Version) Before(major int) bool { return v.Major() < major }

// String returns "major.minor", e.g. "17.11".
func (v Version) String() string {
	return strconv.Itoa(v.Major()) + "." + strconv.Itoa(v.Minor())
}

// versioned holds the variants of one query keyed by the first Postgres major they apply to.
type versioned map[int]string

// For returns the variant with the highest key that is not newer than the server major. ok is false when
// the server is older than every variant.
func (q versioned) For(v Version) (string, bool) {
	keys := slices.Sorted(maps.Keys(q))
	for _, k := range slices.Backward(keys) {
		if k <= v.Major() {
			return q[k], true
		}
	}
	return "", false
}
