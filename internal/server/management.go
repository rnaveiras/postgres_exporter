package server

// Management endpoints under /-/, following the Prometheus management API
// (https://prometheus.io/docs/prometheus/latest/management_api/). /-/healthy and /-/ready are always served
// and never query Postgres, so probes do not open database connections. /-/log-level changes the exporter
// at runtime and is served only with --web.enable-admin-api.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"slices"
	"strings"
	"sync"
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

// LogLevelNames are the levels accepted by --log.level and /-/log-level, from most to least verbose.
var LogLevelNames = []string{"debug", "info", "warn", "error"}

// errLogLevel lists the accepted names, so the flag and the endpoint report the same choices.
var errLogLevel = errors.New("level must be one of: " + strings.Join(LogLevelNames, ", "))

// ParseLogLevel returns the level for one of LogLevelNames, in any case. It rejects the offsets slog itself
// accepts, such as "info+2".
func ParseLogLevel(name string) (slog.Level, error) {
	if !slices.Contains(LogLevelNames, strings.ToLower(name)) {
		return 0, errLogLevel
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(name)); err != nil {
		return 0, errLogLevel
	}
	return level, nil
}

func levelName(l slog.Level) string {
	return strings.ToLower(l.String())
}

// healthyHandler answers 200 while the process serves HTTP. It does not check Postgres: postgres_up
// reports that, and a liveness probe that failed with Postgres would restart a healthy exporter.
func healthyHandler(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeText(logger, w, "Postgres Exporter is Healthy.\n")
	})
}

// readyHandler answers 200 whenever the process serves HTTP. The exporter has no startup work to wait for,
// and the listener only accepts connections while serving, so there is no not-ready state a probe could see.
// It does not check Postgres: a readiness probe that failed with Postgres would stop scrapes, and with them
// postgres_up 0.
func readyHandler(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeText(logger, w, "Postgres Exporter is Ready.\n")
	})
}

// allowMethods answers other methods with 405 and the Allow header instead of letting them fall through to
// the catch-all 404.
func allowMethods(h http.Handler, methods ...string) http.Handler {
	allow := strings.Join(methods, ", ")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slices.Contains(methods, r.Method) {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Allow", allow)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func writeText(logger *slog.Logger, w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
	return c.stateLocked()
}

func (c *logLevelControl) stateLocked() logLevelState {
	s := logLevelState{Level: levelName(c.level.Level())}
	if !c.revertAt.IsZero() {
		at := c.revertAt.UTC()
		s.RevertTo = levelName(c.base)
		s.RevertAt = &at
	}
	return s
}

// set changes the level, for d when d > 0 and permanently otherwise, and returns the previous level and the
// resulting state, read under the same lock so a concurrent change cannot show up in this response. Any
// pending revert is canceled.
func (c *logLevelControl) set(ctx context.Context, level slog.Level, d time.Duration) (slog.Level, logLevelState) {
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
		return old, c.stateLocked()
	}
	gen := c.gen
	c.revertAt = time.Now().Add(d)
	// The revert outlives the request that scheduled it.
	revertCtx := context.WithoutCancel(ctx)
	c.timer = time.AfterFunc(d, func() { c.revert(revertCtx, gen) })
	return old, c.stateLocked()
}

func (c *logLevelControl) revert(ctx context.Context, gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if gen != c.gen {
		return // a newer change replaced this one
	}
	old := c.level.Level()
	c.level.Set(c.base)
	c.timer = nil
	c.revertAt = time.Time{}
	c.logger.Log(ctx, auditLevel(old, c.base), "temporary log level expired",
		"from", levelName(old),
		"to", levelName(c.base))
}

// auditLevel is the level a log level change is recorded at: at least warn, and never below the level in
// force before or after the change, so the record is not filtered out by the change it records.
func auditLevel(from, to slog.Level) slog.Level {
	return max(slog.LevelWarn, from, to)
}

// ServeHTTP answers GET and HEAD with the current state, and PUT and POST with a change.
func (c *logLevelControl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		c.writeState(w, c.state())
	case http.MethodPut, http.MethodPost:
		c.change(w, r)
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *logLevelControl) change(w http.ResponseWriter, r *http.Request) {
	// A browser sends text/plain and form bodies cross-origin without a preflight; requiring JSON makes a
	// page on another origin unable to change the level through a visitor's browser.
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil ||
		mediaType != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxLogLevelBody))
	dec.DisallowUnknownFields()
	var req logLevelRequest
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		http.Error(w, "invalid request body: one JSON object expected", http.StatusBadRequest)
		return
	}

	level, err := ParseLogLevel(req.Level)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	old, state := c.set(r.Context(), level, d)
	attrs := []any{
		"from", levelName(old),
		"to", levelName(level),
		"remote_addr", r.RemoteAddr,
	}
	if d > 0 {
		attrs = append(attrs, "for", d.String())
	}
	c.logger.Log(r.Context(), auditLevel(old, level), "log level changed", attrs...)

	c.writeState(w, state)
}

func (c *logLevelControl) writeState(w http.ResponseWriter, state logLevelState) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		c.logger.Debug("write response", slog.Any(errorKey, err))
	}
}
