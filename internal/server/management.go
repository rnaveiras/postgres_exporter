package server

// Management endpoints under /-/, following the Prometheus management API
// (https://prometheus.io/docs/prometheus/latest/management_api/). /-/healthy and /-/ready are always served
// and never query Postgres, so probes do not open database connections. /-/log-level changes the exporter
// at runtime and is served only with --web.enable-admin-api.

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	healthyPath  = "/-/healthy"
	readyPath    = "/-/ready"
	logLevelPath = "/-/log-level"

	// maxLogLevelFor caps how long a temporary log level change lasts.
	maxLogLevelFor = 24 * time.Hour
	// maxLogLevelBody caps the request body of a log level change.
	maxLogLevelBody = 1 << 10
)

// logLevels are the levels accepted by --log.level and /-/log-level.
var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func levelName(l slog.Level) string {
	return strings.ToLower(l.String())
}

// healthyHandler answers 200 while the process serves HTTP. It does not check Postgres: postgres_up
// reports that, and a liveness probe that failed with Postgres would restart a healthy exporter.
func healthyHandler(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeText(logger, w, http.StatusOK, "Postgres Exporter is Healthy.\n")
	})
}

// readyHandler answers 200 once the listener is bound and 503 after shutdown starts. It does not check
// Postgres: a readiness probe that failed with Postgres would stop scrapes, and with them postgres_up 0.
func readyHandler(logger *slog.Logger, ready *atomic.Bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !ready.Load() {
			writeText(logger, w, http.StatusServiceUnavailable, "Service Unavailable\n")
			return
		}
		writeText(logger, w, http.StatusOK, "Postgres Exporter is Ready.\n")
	})
}

func writeText(logger *slog.Logger, w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	if _, err := io.WriteString(w, body); err != nil {
		logger.Debug("write response", slog.Any(errorKey, err))
	}
}

// logLevelControl reads and changes the log level at runtime. A change with a duration is temporary: when it
// expires the level reverts to the last permanent level, so debug logging cannot be left on by accident.
type logLevelControl struct {
	logger *slog.Logger
	level  *slog.LevelVar

	mu sync.Mutex
	// base is the last permanent level, the one a temporary change reverts to.
	base slog.Level
	// revertAt is when the temporary change expires; zero when none is active.
	revertAt time.Time
	timer    *time.Timer
	// gen increases on every change, so a revert timer that fires after a newer change does nothing.
	gen uint64
}

func newLogLevelControl(logger *slog.Logger, level *slog.LevelVar) *logLevelControl {
	return &logLevelControl{logger: logger, level: level, base: level.Level()}
}

// logLevelState is the JSON body of every /-/log-level response.
type logLevelState struct {
	Level    string     `json:"level"`
	RevertTo string     `json:"revert_to,omitempty"`
	RevertAt *time.Time `json:"revert_at,omitempty"`
}

// logLevelRequest is the JSON body of a /-/log-level change. For is a Go duration such as "15m"; empty
// makes the change permanent.
type logLevelRequest struct {
	Level string `json:"level"`
	For   string `json:"for,omitempty"`
}

func (c *logLevelControl) state() logLevelState {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := logLevelState{Level: levelName(c.level.Level())}
	if !c.revertAt.IsZero() {
		at := c.revertAt.UTC()
		s.RevertTo = levelName(c.base)
		s.RevertAt = &at
	}
	return s
}

// set changes the level, for d when d > 0 and permanently otherwise, and returns the previous level. Any
// pending revert is canceled.
func (c *logLevelControl) set(level slog.Level, d time.Duration) slog.Level {
	c.mu.Lock()
	defer c.mu.Unlock()

	old := c.level.Level()
	c.gen++
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
		c.revertAt = time.Time{}
	}
	c.level.Set(level)

	if d <= 0 {
		c.base = level
		return old
	}
	gen := c.gen
	c.revertAt = time.Now().Add(d)
	c.timer = time.AfterFunc(d, func() { c.revert(gen) })
	return old
}

func (c *logLevelControl) revert(gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if gen != c.gen {
		return // a newer change replaced this one
	}
	old := c.level.Level()
	c.level.Set(c.base)
	c.timer = nil
	c.revertAt = time.Time{}
	c.logger.Warn("temporary log level expired",
		"from", levelName(old),
		"to", levelName(c.base))
}

// ServeHTTP answers GET and HEAD with the current state, and PUT and POST with a change.
func (c *logLevelControl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		c.writeState(w)
	case http.MethodPut, http.MethodPost:
		c.change(w, r)
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *logLevelControl) change(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxLogLevelBody))
	dec.DisallowUnknownFields()
	var req logLevelRequest
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	level, ok := logLevels[strings.ToLower(req.Level)]
	if !ok {
		http.Error(w, "level must be one of: debug, info, warn, error", http.StatusBadRequest)
		return
	}

	var d time.Duration
	if req.For != "" {
		var err error
		d, err = time.ParseDuration(req.For)
		if err != nil || d <= 0 || d > maxLogLevelFor {
			http.Error(w, "for must be a duration greater than 0 and at most "+maxLogLevelFor.String(),
				http.StatusBadRequest)
			return
		}
	}

	old := c.set(level, d)
	attrs := []any{
		"from", levelName(old),
		"to", levelName(level),
		"remote_addr", r.RemoteAddr,
	}
	if d > 0 {
		attrs = append(attrs, "for", d.String())
	}
	c.logger.Warn("log level changed", attrs...)

	c.writeState(w)
}

func (c *logLevelControl) writeState(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(c.state()); err != nil {
		c.logger.Debug("write response", slog.Any(errorKey, err))
	}
}
