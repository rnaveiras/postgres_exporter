// Package integration runs the exporter against a real Postgres server.
//
// It is a separate Go module because it depends on the private github.com/rnaveiras/pgfresh module;
// keeping that dependency out of the main module means building and unit-testing the exporter never
// needs access to it.
//
// The server comes from pgfresh: a container started from POSTGRES_IMAGE, or an existing server when
// PGFRESH_URL is set. Run it once per Postgres major.
package integration

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	pgx "github.com/jackc/pgx/v5"
	"github.com/rnaveiras/pgfresh"
	"github.com/rnaveiras/pgfresh/pgxconn"
)

//go:embed testdata/seed.sql
var seedSQL string

var (
	h        *pgfresh.Handle[pgx.Tx]
	pgMajor  int
	pgNum    string
	pgFull   string
	adminDSN string
)

func TestMain(m *testing.M) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	teardown, err := setup(context.Background())
	if err != nil {
		logger.Error("integration setup failed", "error", err)
		os.Exit(1) //nolint:revive // setup failed before m.Run; exit non-zero rather than report zero tests
	}
	defer teardown()

	logger.Info("integration tests",
		"server_version", pgFull,
		"image", os.Getenv("POSTGRES_IMAGE"))
	m.Run()
}

// setup starts or attaches to the Postgres server, creates the seeded template and records the server version.
func setup(ctx context.Context) (teardown func(), err error) {
	// Reuse keeps the template between runs and rebuilds it only when the seed SQL changes; without it a
	// second run against an existing server (PGFRESH_URL) fails on the leftover template.
	opts := []pgfresh.Option{pgfresh.WithReuse(true), pgfresh.WithPostgresSettings(map[string]string{
		"shared_preload_libraries": "pg_stat_statements",
		"track_io_timing":          "on",
		"track_functions":          "all",
	})}
	if image := os.Getenv("POSTGRES_IMAGE"); image != "" {
		// pgfresh names its container after the worktree only, so each major needs its own ID or a run
		// could attach to a container left over from another major.
		if os.Getenv("PGFRESH_ID") == "" && os.Getenv("PGFRESH_URL") == "" {
			return nil, errors.New("POSTGRES_IMAGE is set but PGFRESH_ID is not: set a PGFRESH_ID unique to this Postgres major")
		}
		opts = append(opts, pgfresh.WithImage(image))
	}

	// The monitoring role is the one operators are told to create; tests connect as it, never as superuser.
	role, err := os.ReadFile(filepath.Join("..", "docker", "sql", "monitoring-role.sql"))
	if err != nil {
		return nil, fmt.Errorf("read monitoring role: %w", err)
	}

	h, err = pgfresh.Setup(ctx, pgfresh.Database{Name: "exporter", SQL: seedSQL + "\n" + string(role)}, pgxconn.New(), opts...)
	if err != nil {
		return nil, fmt.Errorf("pgfresh setup: %w", err)
	}
	teardown = func() { h.Teardown(context.WithoutCancel(ctx)) }
	adminDSN = h.ConnString()

	conn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		teardown()
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	if err := conn.QueryRow(ctx, `SELECT current_setting('server_version_num'), current_setting('server_version')`).Scan(&pgNum, &pgFull); err != nil {
		teardown()
		return nil, fmt.Errorf("server version: %w", err)
	}
	n, err := strconv.Atoi(pgNum)
	if err != nil {
		teardown()
		return nil, fmt.Errorf("server_version_num %q: %w", pgNum, err)
	}
	pgMajor = n / 10000
	return teardown, nil
}
