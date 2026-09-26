package main

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/tracelog"
)

const (
	pgxLogMessage = "pgx log"
)

// pgxToSlogLevel maps pgx log levels to slog levels. Levels missing from the map, including any added
// by a newer pgx, log at slog.LevelInfo.
var pgxToSlogLevel = map[tracelog.LogLevel]slog.Level{
	tracelog.LogLevelTrace: slog.LevelDebug,
	tracelog.LogLevelDebug: slog.LevelDebug,
	tracelog.LogLevelInfo:  slog.LevelInfo,
	tracelog.LogLevelWarn:  slog.LevelWarn,
	tracelog.LogLevelError: slog.LevelError,
	tracelog.LogLevelNone:  slog.LevelInfo,
}

// SlogAdapter adapts slog to pgx logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

func (s *SlogAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	slogLevel, ok := pgxToSlogLevel[level]
	if !ok {
		slogLevel = slog.LevelInfo
	}

	s.logger.LogAttrs(ctx, slogLevel, pgxLogMessage,
		slog.Group("pgx",
			slog.String("msg", msg),
			slog.Any("data", data)))
}
