package collector

import (
	"maps"
	"slices"
	"strconv"
)

const currentDatabaseQuery = `SELECT current_database() /*postgres_exporter*/`

// Query is one SQL statement the exporter runs, with the Postgres majors it applies to.
type Query struct {
	Name string
	SQL  string
	// MinMajor is the first major the statement applies to.
	MinMajor int
	// MaxMajor is the first major the statement no longer applies to; 0 means no upper bound.
	MaxMajor int
}

// AppliesTo reports whether the statement is run against a server of version v.
func (q Query) AppliesTo(v Version) bool {
	return v.Major() >= q.MinMajor && (q.MaxMajor == 0 || v.Major() < q.MaxMajor)
}

// AllQueries lists every SQL statement the exporter can run, one entry per version variant, so tests can
// prepare each of them against every supported major.
func AllQueries() []Query {
	base := []Query{
		{Name: "version", SQL: versionQuery},
		{Name: "platform", SQL: platformQuery},
		{Name: "list_databases", SQL: listDatnameQuery},
		{Name: "current_database", SQL: currentDatabaseQuery},
		{Name: "info_is_in_recovery", SQL: isInRecoveryQuery},
		{Name: "info_is_in_backup", SQL: isInBackupQuery, MaxMajor: isInBackupRemovedVersion},
		{Name: "info_start_time", SQL: startTimeQuery},
		{Name: "info_config_load_time", SQL: configLoadTimeQuery},
		{Name: "locks", SQL: locksQuery},
		{Name: "stat_activity", SQL: statActivityQuery},
		{Name: "stat_activity_oldest_xact", SQL: statActivityScraperXactQuery},
		{Name: "stat_activity_oldest_backend", SQL: statActivityScraperBackendStartQuery},
		{Name: "stat_activity_oldest_active", SQL: statActivityScraperActiveQuery},
		{Name: "stat_activity_oldest_snapshot", SQL: statActivityScraperOldestSnapshotQuery},
		{Name: "stat_activity_oldest_xmin", SQL: statActivityScraperOldestSnapshotXidQuery},
		{Name: "stat_archiver", SQL: statArchiver},
		{Name: "stat_database", SQL: statDatabaseQuery},
		{Name: "stat_replication", SQL: statReplicationLagBytes},
		{Name: "stat_user_tables", SQL: statUserTablesQuery},
		{Name: "stat_user_indexes", SQL: statUserIndexesQuery},
		{Name: "disk_usage_tables", SQL: tableUsageQuery},
		{Name: "disk_usage_indexes", SQL: indexUsageQuery},
	}
	vers := []struct {
		name string
		q    versioned
	}{
		{"stat_bgwriter", statBgwriterQueries},
		{"stat_checkpointer", statCheckpointerQueries},
		{"stat_progress_vacuum", statVacuumProgressQueries},
	}

	queries := make([]Query, 0, len(base)+len(vers)*2)
	for _, q := range base {
		if q.MinMajor == 0 {
			q.MinMajor = MinSupportedVersion
		}
		queries = append(queries, q)
	}
	for _, v := range vers {
		queries = append(queries, variants(v.name, v.q)...)
	}
	return queries
}

// variants expands a versioned table into one Query per variant, each valid until the next variant starts.
func variants(name string, q versioned) []Query {
	keys := slices.Sorted(maps.Keys(q))
	out := make([]Query, 0, len(keys))
	for i, k := range keys {
		next := 0
		if i+1 < len(keys) {
			next = keys[i+1]
		}
		out = append(out, Query{Name: name + "_pg" + strconv.Itoa(k), SQL: q[k], MinMajor: k, MaxMajor: next})
	}
	return out
}
