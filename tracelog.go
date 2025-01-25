package main

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/tracelog"
)

// SlogAdapter adapts slog to pgx logger interface
type SlogAdapter struct {
	logger *slog.Logger
}

func (s *SlogAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	var slogLevel slog.Level

	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		slogLevel = slog.LevelDebug
	case tracelog.LogLevelInfo:
		slogLevel = slog.LevelInfo
	case tracelog.LogLevelWarn:
		slogLevel = slog.LevelWarn
	case tracelog.LogLevelError:
		slogLevel = slog.LevelError
	}

	s.logger.LogAttrs(ctx, slogLevel, msg, slog.Any("pgx_data", data))
}
