// Command postgres_exporter serves Prometheus metrics for a PostgreSQL server.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/common/version"

	"github.com/rnaveiras/postgres_exporter/internal/server"
)

// defaultExcludedDatabases are administrative databases of managed Postgres services that the monitoring
// role usually cannot connect to.
var defaultExcludedDatabases = []string{"cloudsqladmin", "rdsadmin", "azure_maintenance", "azure_sys"}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err) //nolint:revive // exiting anyway, a failed stderr write changes nothing
		os.Exit(1)
	}
}

// run parses the command line, sets up logging and serves until ctx is canceled.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, a, err := parseFlags(args, stdout)
	if err != nil {
		a.Usage(args)
		return fmt.Errorf("parse command line arguments: %w", err)
	}

	logLevel := new(slog.LevelVar)
	logger, err := setupLogger(stderr, logLevel, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		return err
	}

	logger.Info("starting postgres exporter", "version", version.Info())
	logger.Info("build context", "build_context", version.BuildContext())
	logger.Debug("cfg", "cfg", cfg)

	return server.Run(ctx, cfg.Config, logger, logLevel)
}

// flagConfig is the command line: the server configuration plus the options that only the command uses.
type flagConfig struct {
	server.Config

	LogLevel  string
	LogFormat string
}

// LogValue implements the slog.LogValuer interface. The data source is left out: it may hold a password.
func (f flagConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("listen_address", f.ListenAddress),
		slog.String("metrics_path", f.MetricsPath),
		slog.String("log_level", f.LogLevel),
		slog.String("log_format", f.LogFormat),
		slog.Bool("pprof", f.Pprof),
		slog.Bool("admin_api", f.AdminAPI),
		slog.Bool("legacy_names", f.LegacyNames),
		slog.Any("exclude_databases", f.ExcludedDatabases),
	)
}

// newApp defines the command line flags and binds them to cfg.
func newApp(cfg *flagConfig, usage io.Writer) *kingpin.Application {
	a := kingpin.New(filepath.Base(os.Args[0]), "The Postgres Exporter").UsageWriter(usage)
	a.Version(version.Print("postgres_exporter"))
	a.HelpFlag.Short('h')

	a.Flag("web.listen-address", "Address on which to expose metrics and web interface.").
		Default("0.0.0.0:9187").StringVar(&cfg.ListenAddress)

	a.Flag("web.telemetry-path", "Path under which to expose metrics").
		Default("/metrics").StringVar(&cfg.MetricsPath)

	a.Flag("db.data-source", "libpq compatible connection string, e.g `user=postgres host=/var/run/postgresql`. Leave blank for libqp envs").
		StringVar(&cfg.DataSource)

	a.Flag("db.excluded-databases", "Repeat this flag for each database to exclude from monitoring").
		Default(defaultExcludedDatabases...).StringsVar(&cfg.ExcludedDatabases)

	a.Flag("log.level", "Only log messages with the given severity or above. One of: ["+
		strings.Join(server.LogLevelNames, ", ")+"]").
		Default("info").EnumVar(&cfg.LogLevel, server.LogLevelNames...)

	a.Flag("log.format", "Output format of log messages. One of: [logfmt, json]").
		Default("logfmt").EnumVar(&cfg.LogFormat, "logfmt", "json")

	a.Flag("web.enable-pprof", "Serve the Go runtime profiling endpoints under /debug/pprof/.").
		Default("false").BoolVar(&cfg.Pprof)

	a.Flag("web.enable-admin-api", "Serve /-/log-level, which reads and changes the log level at runtime. "+
		"It has no authentication: enable it only where the listen address is trusted.").
		Default("false").BoolVar(&cfg.AdminAPI)

	a.Flag("compat.legacy-names", "Also emit deprecated metric names next to their replacements.").
		Default("true").BoolVar(&cfg.LegacyNames)

	return a
}

// parseFlags parses args into a flagConfig.
func parseFlags(args []string, usage io.Writer) (flagConfig, *kingpin.Application, error) {
	cfg := flagConfig{}
	a := newApp(&cfg, usage)
	_, err := a.Parse(args)
	return cfg, a, err
}

// setupLogger configures the logger.
func setupLogger(w io.Writer, logLevelVar *slog.LevelVar, logFormat, logLevel string) (*slog.Logger, error) {
	level, err := server.ParseLogLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}
	logLevelVar.Set(level)

	handlerOpts := slog.HandlerOptions{
		Level:     logLevelVar,
		AddSource: false,
	}

	var handler slog.Handler
	if logFormat == "logfmt" {
		handler = slog.NewTextHandler(w, &handlerOpts)
	} else {
		handler = slog.NewJSONHandler(w, &handlerOpts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}
