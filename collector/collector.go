package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/jackc/pgx"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// Namespace defines the common namespace to be used by all metrics.
	namespace = "postgres"
	// Subsystem(s)
	exporter = "exporter"
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"postgres_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"postgres_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

const (
	defaultEnabled  = true
	defaultDisabled = false
)

var (
	factories      = make(map[string]func() (Collector, error))
	collectorState = make(map[string]*bool)
)

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ctx context.Context, db *pgx.Conn, ch chan<- prometheus.Metric) error
}

func registerCollector(collector string, isDefaultEnabled bool, factory func() (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", strings.Replace(collector, "_", "-", -1))
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s).", collector, helpDefaultState)
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)

	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Bool()
	collectorState[collector] = flag
	factories[collector] = factory
}

// PostgresCollector implements the prometheus.Collector interface.
type postgresCollector struct {
	ctx        context.Context
	db         *pgx.Conn
	Collectors map[string]Collector
	logger     log.Logger
}

// NewPostgresCollector creates a new postgresCollector
func NewPostgresCollector(ctx context.Context, db *pgx.Conn, logger log.Logger, filters ...string) (*postgresCollector, error) {
	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}
		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}
		f[filter] = true
	}
	collectors := make(map[string]Collector)
	for key, enabled := range collectorState {
		if *enabled {
			collector, err := factories[key]()
			if err != nil {
				return nil, err
			}
			if len(f) == 0 || f[key] {
				collectors[key] = collector
			}
		}
	}
	return &postgresCollector{
		ctx:        ctx,
		db:         db,
		logger:     logger,
		Collectors: collectors}, nil
}

// Describe implements the prometheus.Collector interface.
func (n postgresCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n postgresCollector) Collect(ch chan<- prometheus.Metric) {
	for name, c := range n.Collectors {
		n.execute(name, c, ch)
	}
}

func (n postgresCollector) execute(name string, c Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(n.ctx, n.db, ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		level.Debug(n.logger).Log("collector", name, "duration", duration.Seconds(), "error", err)
		success = 0
	} else {
		level.Debug(n.logger).Log("collector", name, "duration", duration.Seconds(), "event", "collector.success")
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}
