package collector

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	versionQuery = `
SELECT current_setting('server_version_num')
     , current_setting('server_version') /*postgres_exporter*/`
	listDatnameQuery = `
SELECT datname FROM pg_database
WHERE datallowconn = true AND datistemplate = false
AND datname != ALL($1) /*postgres_exporter*/`
	successValue    = 1.0
	failureValue    = 0.0
	infoMetricValue = 1.0
	errorKey        = "error"

	// connectScraperName labels the per-database connection attempt in the scraper metrics.
	connectScraperName = "connect"
	// listDatabasesScraperName labels the database list in the scraper metrics. Without it every
	// per-database series vanishes, so its failure has to show up as a failed scraper.
	listDatabasesScraperName = "list_databases"
	// unsupportedVersionWarnInterval limits how often a scrape of an unsupported server logs a warning.
	unsupportedVersionWarnInterval = time.Hour
)

var (
	upDesc = prometheus.NewDesc(
		"postgres_up",
		"Whether the Postgres server is up.",
		nil,
		nil,
	)
	infoDesc = prometheus.NewDesc(
		"postgres_info",
		"Postgres server information: server_version, server_version_num and the detected platform.",
		[]string{"version", "version_num", "platform"},
		nil,
	)
	unsupportedVersionDesc = prometheus.NewDesc(
		"postgres_exporter_unsupported_version",
		"1 when the Postgres major is older than the oldest supported major ("+strconv.Itoa(MinSupportedVersion)+").",
		nil,
		nil,
	)
	untestedVersionDesc = prometheus.NewDesc(
		"postgres_exporter_untested_version",
		"1 when the Postgres major is newer than the newest tested major ("+strconv.Itoa(MaxTestedVersion)+").",
		nil,
		nil,
	)
	scrapeDurationDesc = prometheus.NewDesc(
		"postgres_exporter_scraper_duration_seconds",
		"Duration of a scrapers scrape.",
		[]string{"scraper", labelDatname},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		"postgres_exporter_scraper_success",
		"Whether a scraper succeeded.",
		[]string{"scraper", labelDatname},
		nil,
	)

	// versionLog rate-limits the version support log lines across scrapes.
	versionLog = &versionLogger{last: map[int]time.Time{}}
)

// Scraper is the interface each scraper has to implement.
type Scraper interface {
	Name() string
	// Scrape new metrics and expose them via prometheus registry.
	Scrape(ctx context.Context, db Querier, version Version, ch chan<- prometheus.Metric) error
}

// Options configures the scrapers.
type Options struct {
	// ExcludedDatabases are never scraped by the per-database scrapers.
	ExcludedDatabases []string
	// LegacyNames also emits deprecated metric names next to their replacements.
	LegacyNames bool
}

type Exporter struct {
	ctx               context.Context //nolint:containedctx // removed when the context is passed to Collect instead of stored
	logger            *slog.Logger
	connConfig        *pgx.ConnConfig
	scrapers          []Scraper
	datnameScrapers   []Scraper
	excludedDatabases []string
}

// Verify our Exporter satisfies the prometheus.Collector interface.
var _ prometheus.Collector = (*Exporter)(nil)

// NewExporter is called every time we receive a scrape request and knows how
// to collect metrics using each of the scrapers. It will live only for the
// duration of the scrape request. connConfig is never modified.
func NewExporter(ctx context.Context, logger *slog.Logger, connConfig *pgx.ConnConfig, opts Options) *Exporter {
	return &Exporter{
		ctx:        ctx,
		logger:     logger,
		connConfig: connConfig,
		scrapers: []Scraper{
			NewInfoScraper(),
			NewLocksScraper(),
			NewStatActivityScraper(),
			NewStatArchiverScraper(),
			NewStatBgwriterScraper(),
			NewStatCheckpointerScraper(),
			NewStatDatabaseScraper(),
			NewStatReplicationScraper(),
		},
		datnameScrapers: []Scraper{
			NewStatVacuumProgressScraper(),
			NewStatUserTablesScraper(opts.LegacyNames),
			NewStatUserIndexesScraper(),
			NewDiskUsageScraper(),
		},
		excludedDatabases: opts.ExcludedDatabases,
	}
}

// Describe implements the prometheus.Collector interface.
func (*Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	conn, err := pgx.ConnectConfig(e.ctx, e.connConfig)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, failureValue)
		e.logger.Error("exporter collect",
			slog.Any(errorKey, err))
		return // cannot continue without a valid connection
	}
	defer e.close(conn)

	e.collectFrom(conn, ch)
}

// collectFrom scrapes the server over an open connection. postgres_up is 1 only once the server answers the
// version query: a server that accepts connections but cannot run queries is not up.
func (e *Exporter) collectFrom(conn Querier, ch chan<- prometheus.Metric) {
	v, err := queryVersion(e.ctx, conn)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, failureValue)
		e.logger.Error("version query",
			slog.Any(errorKey, err))
		return // cannot continue without a version
	}

	// postgres_up
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, successValue)

	platform, err := DetectPlatform(e.ctx, conn)
	if err != nil {
		e.logger.Debug("platform detection failed, assuming community postgres",
			slog.Any(errorKey, err))
	}

	// postgres_info
	ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, infoMetricValue,
		v.Full, strconv.Itoa(v.Num), string(platform))
	e.reportVersionSupport(v, ch)

	start := time.Now()
	dbnames, err := e.listDatabases(conn)
	e.report(listDatabasesScraperName, e.connConfig.Database, time.Since(start), err, ch)

	// run global scrapers
	for _, scraper := range e.scrapers {
		e.scrape(scraper, conn, v, e.connConfig.Database, ch)
	}

	// run datname scrapers
	for _, dbname := range dbnames {
		e.scrapeDatabase(dbname, v, ch)
	}
}

// scrapeDatabase runs the per-database scrapers on their own connection. A failed connection is reported
// as scraper "connect" for that database and does not stop the other databases.
func (e *Exporter) scrapeDatabase(dbname string, v Version, ch chan<- prometheus.Metric) {
	cfg := e.connConfig.Copy()
	cfg.Database = dbname

	start := time.Now()
	conn, err := pgx.ConnectConfig(e.ctx, cfg)
	e.report(connectScraperName, dbname, time.Since(start), err, ch)
	if err != nil {
		return
	}
	defer e.close(conn)

	for _, scraper := range e.datnameScrapers {
		e.scrape(scraper, conn, v, dbname, ch)
	}
}

func (e *Exporter) listDatabases(conn Querier) ([]string, error) {
	e.logger.Debug("excluded databases",
		slog.String("databases", strings.Join(e.excludedDatabases, ",")))

	// A nil slice is sent as NULL, and `datname != ALL(NULL)` matches no database at all.
	excluded := e.excludedDatabases
	if excluded == nil {
		excluded = []string{}
	}
	rows, err := conn.Query(e.ctx, listDatnameQuery, excluded)
	if err != nil {
		return nil, err
	}
	dbnames, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	e.logger.Debug("databases found",
		slog.String("databases", strings.Join(dbnames, ",")))
	return dbnames, nil
}

func (e *Exporter) scrape(scraper Scraper, conn Querier, version Version, datname string, ch chan<- prometheus.Metric) {
	start := time.Now()
	err := scraper.Scrape(e.ctx, conn, version, ch)
	if errors.Is(err, ErrUnsupportedVersion) {
		e.logger.Debug("scraper skipped",
			"scraper", scraper.Name(),
			"version", version.Full)
		err = nil
	}
	e.report(scraper.Name(), datname, time.Since(start), err, ch)
}

func (e *Exporter) report(name, datname string, duration time.Duration, err error, ch chan<- prometheus.Metric) {
	success := successValue
	if err != nil {
		e.logger.Error("failed scrape",
			"scraper", name,
			labelDatname, datname,
			"duration", duration.Seconds(),
			errorKey, err)
		success = failureValue
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name, datname)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name, datname)
}

// reportVersionSupport emits the version support gauges and logs, rate-limited, when the server is outside
// the tested range. Scraping always continues.
func (e *Exporter) reportVersionSupport(v Version, ch chan<- prometheus.Metric) {
	unsupported, untested := failureValue, failureValue

	switch {
	case v.Before(MinSupportedVersion):
		unsupported = successValue
		if versionLog.due(v.Major(), unsupportedVersionWarnInterval) {
			e.logger.Warn("postgres version is not supported, scraping on a best-effort basis",
				"version", v.Full,
				"min_supported", MinSupportedVersion)
		}
	case v.Major() > MaxTestedVersion:
		untested = successValue
		if versionLog.due(v.Major(), 0) {
			e.logger.Info("postgres version is newer than the newest tested version",
				"version", v.Full,
				"max_tested", MaxTestedVersion)
		}
	default:
		// within the supported and tested range
	}

	ch <- prometheus.MustNewConstMetric(unsupportedVersionDesc, prometheus.GaugeValue, unsupported)
	ch <- prometheus.MustNewConstMetric(untestedVersionDesc, prometheus.GaugeValue, untested)
}

func (e *Exporter) close(conn *pgx.Conn) {
	if err := conn.Close(e.ctx); err != nil {
		e.logger.Debug("close connection",
			slog.Any(errorKey, err))
	}
}

func queryVersion(ctx context.Context, db Querier) (Version, error) {
	var num, full string
	if err := db.QueryRow(ctx, versionQuery).Scan(&num, &full); err != nil {
		return Version{}, err
	}
	return ParseVersion(num, full)
}

// versionLogger remembers when each major was last logged. An interval of 0 logs once per process.
type versionLogger struct {
	mu   sync.Mutex
	last map[int]time.Time
}

func (l *versionLogger) due(major int, interval time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	last, seen := l.last[major]
	if seen && (interval == 0 || time.Since(last) < interval) {
		return false
	}
	l.last[major] = time.Now()
	return true
}
