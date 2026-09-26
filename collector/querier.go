package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v5"
)

// Querier is the part of *pgx.Conn the scrapers use. Unit tests substitute a mock.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var _ Querier = (*pgx.Conn)(nil)
