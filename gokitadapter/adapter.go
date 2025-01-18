package gokitadapter

import (
	"context"
	"fmt"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	pgx "github.com/jackc/pgx/v4"
)

const maxSQLLength = 32

// Logger ...
type Logger struct {
	logger kitlog.Logger
}

// NewLogger ...
func NewLogger(logger kitlog.Logger) *Logger {
	return &Logger{logger: kitlog.With(logger, "component", "pgx")}
}

// Log (pgx compatible)
func (l *Logger) Log(_ context.Context, logLevel pgx.LogLevel, msg string, data map[string]any) {
	fieldsLogger := l.logger

	for key, value := range data {
		fieldsLogger = processLogField(fieldsLogger, key, value)
	}

	pgxLogLevel(logLevel)(fieldsLogger).Log("msg", msg)
}

func processLogField(logger kitlog.Logger, key string, value any) kitlog.Logger {
	switch key {
	case "args":
		// Deliberately skips logging SQL arguments for security/privacy
		return logger

	case "time":
		v, ok := value.(time.Duration)
		if !ok {
			return kitlog.With(logger, "duration_error",
				fmt.Sprintf("invalid duration type: %T", value))
		}
		return kitlog.With(logger, "duration", v.Seconds())

	case "sql":
		v, ok := value.(string)
		if !ok {
			return kitlog.With(logger, key,
				fmt.Sprintf("invalid SQL value type %T", value))
		}
		if len(v) > maxSQLLength {
			return kitlog.With(logger, key,
				fmt.Sprintf("%s (truncated %d bytes)", v[:maxSQLLength], len(v)-maxSQLLength))
		}
		return kitlog.With(logger, key, value)

	default:
		return kitlog.With(logger, key, value)
	}
}

func pgxLogLevel(l any) func(kitlog.Logger) kitlog.Logger {
	switch l {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		return level.Debug
	case pgx.LogLevelInfo:
		return level.Info
	case pgx.LogLevelWarn:
		return level.Warn
	case pgx.LogLevelError:
		return level.Error
	default:
		return level.Error
	}
}
