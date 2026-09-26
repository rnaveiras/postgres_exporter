package collector

// Label names shared by several scrapers.
const (
	labelDatname    = "datname"
	labelSchemaname = "schemaname"
	labelRelname    = "relname"
	labelIndexname  = "indexname"
)

// millisecondsPerSecond converts the millisecond timings in pg_stat_* views to seconds.
const millisecondsPerSecond = 1000.0
