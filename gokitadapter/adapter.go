package gokitadapter

import (
	"fmt"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/jackc/pgx"
)

// Logger ...
type Logger struct {
	logger kitlog.Logger
}

// NewLogger ...
func NewLogger(logger kitlog.Logger) *Logger {
	return &Logger{logger: kitlog.With(logger, "component", "pgx")}
}

// Log (pgx compatible)
func (l *Logger) Log(level pgx.LogLevel, msg string, data map[string]interface{}) {
	fieldsLogger := l.logger

	for key, value := range data {
		switch key {
		case "args":
			break

		case "time":
			v := value.(time.Duration)
			fieldsLogger = kitlog.With(fieldsLogger, "duration", v.Seconds())
			break

		case "sql":
			v := value.(string)
			if len(v) > 32 {
				fieldsLogger = kitlog.With(fieldsLogger, key, fmt.Sprintf("%s (truncated %d bytes)", v[:32], len(v)-32))
			} else {
				fieldsLogger = kitlog.With(fieldsLogger, key, value)
			}
		default:
			fieldsLogger = kitlog.With(fieldsLogger, key, value)
		}
	}

	pgxLogLevel(level)(fieldsLogger).Log("msg", msg)
}

func pgxLogLevel(l interface{}) func(kitlog.Logger) kitlog.Logger {
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
