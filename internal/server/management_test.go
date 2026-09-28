package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_probeHandlers_answerWithoutPostgres(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	tests := []struct {
		name    string
		handler http.Handler
		body    string
	}{
		{name: "healthy says the process serves", handler: healthyHandler(logger), body: "Postgres Exporter is Healthy.\n"},
		{name: "ready says the process serves", handler: readyHandler(logger), body: "Postgres Exporter is Ready.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := request(t, tt.handler, http.MethodGet, "", "")
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
			assert.Equal(t, tt.body, rec.Body.String())
		})
	}
}

func TestServer_ParseLogLevel_acceptsTheFlagNames(t *testing.T) {
	t.Parallel()

	for _, name := range LogLevelNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			level, err := ParseLogLevel(strings.ToUpper(name))
			require.NoError(t, err)
			assert.Equal(t, name, levelName(level))
		})
	}
}

func TestServer_ParseLogLevel_rejectsOtherNames(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"trace", "info+2", ""} {
		t.Run("rejects "+name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseLogLevel(name)
			require.ErrorContains(t, err, "debug, info, warn, error")
		})
	}
}

func TestServer_logLevelControl_ServeHTTP_getReturnsCurrentLevel(t *testing.T) {
	t.Parallel()

	c, _ := newTestLogLevelControl(slog.LevelInfo)
	// reading twice and then changing checks that a read releases the lock.
	for range 2 {
		rec := request(t, c, http.MethodGet, "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"level":"info"}`, rec.Body.String())
	}
	require.Equal(t, http.StatusOK, request(t, c, http.MethodPut, "application/json", `{"level":"warn"}`).Code)
}

func TestServer_allowMethods_servesOnlyTheAllowedMethods(t *testing.T) {
	t.Parallel()

	h := allowMethods(healthyHandler(slog.New(slog.DiscardHandler)), http.MethodGet, http.MethodHead)

	rec := request(t, h, http.MethodGet, "", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Postgres Exporter is Healthy.\n", rec.Body.String())

	rec = request(t, h, http.MethodPost, "", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Equal(t, "GET, HEAD", rec.Header().Get("Allow"))
}

func TestServer_logLevelControl_ServeHTTP_acceptsChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		body   string
		want   slog.Level
	}{
		{name: "put sets the level", method: http.MethodPut, body: `{"level":"debug"}`, want: slog.LevelDebug},
		{name: "post sets the level", method: http.MethodPost, body: `{"level":"warn"}`, want: slog.LevelWarn},
		{name: "level names are case insensitive", method: http.MethodPut, body: `{"level":"ERROR"}`, want: slog.LevelError},
		{name: "a day is the longest temporary change", method: http.MethodPut, body: `{"level":"debug","for":"24h"}`, want: slog.LevelDebug},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, _ := newTestLogLevelControl(slog.LevelInfo)
			rec := request(t, c, tt.method, "application/json", tt.body)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, tt.want, c.level.Level())
			assert.Equal(t, levelName(tt.want), decodeState(t, rec).Level)
		})
	}
}

func TestServer_logLevelControl_ServeHTTP_rejectsChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		body        string
		want        int
	}{
		{name: "unknown levels are rejected", body: `{"level":"trace"}`, want: http.StatusBadRequest},
		{name: "slog offset levels are rejected", body: `{"level":"info+2"}`, want: http.StatusBadRequest},
		{name: "a missing level is rejected", body: `{}`, want: http.StatusBadRequest},
		{name: "unknown fields are rejected", body: `{"level":"debug","x":1}`, want: http.StatusBadRequest},
		{name: "a body that is not json is rejected", body: `debug`, want: http.StatusBadRequest},
		{name: "a second object is rejected", body: `{"level":"debug"}{"level":"error"}`, want: http.StatusBadRequest},
		{name: "trailing garbage is rejected", body: `{"level":"debug"} junk`, want: http.StatusBadRequest},
		{name: "an unparsable duration is rejected", body: `{"level":"debug","for":"soon"}`, want: http.StatusBadRequest},
		{name: "a zero duration is rejected", body: `{"level":"debug","for":"0s"}`, want: http.StatusBadRequest},
		{name: "a negative duration is rejected", body: `{"level":"debug","for":"-1m"}`, want: http.StatusBadRequest},
		{name: "more than a day is rejected", body: `{"level":"debug","for":"25h"}`, want: http.StatusBadRequest},
		{
			name: "an oversized body is rejected",
			body: `{"level":"debug","for":"` + strings.Repeat("1", maxLogLevelBody) + `s"}`,
			want: http.StatusBadRequest,
		},
		// a browser sends these cross-origin without a preflight, so they must not change anything.
		{name: "text/plain is refused", contentType: "text/plain", body: `{"level":"debug"}`, want: http.StatusUnsupportedMediaType},
		{
			name: "form encoding is refused", contentType: "application/x-www-form-urlencoded", body: `{"level":"debug"}`,
			want: http.StatusUnsupportedMediaType,
		},
		{name: "a missing content type is refused", contentType: "-", body: `{"level":"debug"}`, want: http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			contentType := tt.contentType
			switch contentType {
			case "":
				contentType = "application/json"
			case "-":
				contentType = ""
			default:
			}
			c, _ := newTestLogLevelControl(slog.LevelInfo)
			rec := request(t, c, http.MethodPut, contentType, tt.body)
			assert.Equal(t, tt.want, rec.Code, rec.Body.String())
			assert.Equal(t, slog.LevelInfo, c.level.Level())
		})
	}
}

func TestServer_logLevelControl_ServeHTTP_advertisesAllowedMethods(t *testing.T) {
	t.Parallel()

	c, _ := newTestLogLevelControl(slog.LevelInfo)
	rec := request(t, c, http.MethodDelete, "", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Equal(t, "GET, HEAD, PUT, POST", rec.Header().Get("Allow"))
}

// the audit line must survive the level it sets: a change to error would otherwise filter out its own record.
func TestServer_logLevelControl_ServeHTTP_logsEveryChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from slog.Level
		body string
		want map[string]string
	}{
		{
			name: "a change to error is logged", from: slog.LevelInfo, body: `{"level":"error"}`,
			want: map[string]string{"from": "info", "to": "error"},
		},
		{
			name: "a change away from error is logged", from: slog.LevelError, body: `{"level":"warn"}`,
			want: map[string]string{"from": "error", "to": "warn"},
		},
		{
			name: "a temporary change logs its duration", from: slog.LevelInfo, body: `{"level":"debug","for":"15m"}`,
			want: map[string]string{"from": "info", "to": "debug", "for": "15m0s"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, logs := newTestLogLevelControl(tt.from)
			require.Equal(t, http.StatusOK, request(t, c, http.MethodPut, "application/json", tt.body).Code)

			rec := logs.find(t, "log level changed")
			for k, v := range tt.want {
				assert.Equal(t, v, rec[k], k)
			}
			assert.Equal(t, "192.0.2.1:1234", rec["remote_addr"])
			if _, temporary := tt.want["for"]; !temporary {
				assert.NotContains(t, rec, "for")
			}
		})
	}
}

// the response reports the state this request produced, taken under the same lock as the change.
func TestServer_logLevelControl_set_returnsTheResultingState(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c, _ := newTestLogLevelControl(slog.LevelInfo)

		old, state := c.set(t.Context(), slog.LevelDebug, 15*time.Minute)
		assert.Equal(t, slog.LevelInfo, old)
		assert.Equal(t, "debug", state.Level)
		assert.Equal(t, "info", state.RevertTo)
		require.NotNil(t, state.RevertAt)
		assert.Equal(t, time.Now().Add(15*time.Minute).UTC(), *state.RevertAt)
	})
}

func TestServer_logLevelControl_set_temporaryRevertsToPermanent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		// starting at error also checks that the expiry line is not filtered out by the level it restores.
		c, logs := newTestLogLevelControl(slog.LevelError)
		require.Equal(t, http.StatusOK,
			request(t, c, http.MethodPut, "application/json", `{"level":"debug","for":"15m"}`).Code)

		time.Sleep(15*time.Minute - time.Second)
		synctest.Wait()
		assert.Equal(t, slog.LevelDebug, c.level.Level())

		time.Sleep(time.Second)
		synctest.Wait()
		assert.Equal(t, slog.LevelError, c.level.Level())
		assert.JSONEq(t, `{"level":"error"}`, request(t, c, http.MethodGet, "", "").Body.String())
		rec := logs.find(t, "temporary log level expired")
		assert.Equal(t, "debug", rec["from"])
		assert.Equal(t, "error", rec["to"])
	})
}

// a permanent change becomes the level later temporary changes revert to.
func TestServer_logLevelControl_set_temporaryRevertsToLatestPermanent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c, _ := newTestLogLevelControl(slog.LevelInfo)
		c.set(t.Context(), slog.LevelWarn, 0)
		_, state := c.set(t.Context(), slog.LevelDebug, 15*time.Minute)
		assert.Equal(t, "warn", state.RevertTo)

		time.Sleep(15 * time.Minute)
		synctest.Wait()
		assert.Equal(t, slog.LevelWarn, c.level.Level())
	})
}

func TestServer_logLevelControl_set_permanentCancelsRevert(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c, _ := newTestLogLevelControl(slog.LevelInfo)
		c.set(t.Context(), slog.LevelDebug, 15*time.Minute)
		_, state := c.set(t.Context(), slog.LevelWarn, 0)
		assert.Equal(t, logLevelState{Level: "warn"}, state, "no revert left pending")

		time.Sleep(time.Hour)
		synctest.Wait()
		assert.Equal(t, slog.LevelWarn, c.level.Level())
	})
}

func TestServer_logLevelControl_set_temporaryKeepsPermanentBase(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c, _ := newTestLogLevelControl(slog.LevelInfo)
		c.set(t.Context(), slog.LevelDebug, 15*time.Minute)
		_, state := c.set(t.Context(), slog.LevelError, time.Hour)
		// the second change still reverts to the permanent level, not to the first temporary one.
		assert.Equal(t, "info", state.RevertTo)

		time.Sleep(30 * time.Minute)
		synctest.Wait()
		assert.Equal(t, slog.LevelError, c.level.Level(), "the first change's timer must not revert")

		time.Sleep(30 * time.Minute)
		synctest.Wait()
		assert.Equal(t, slog.LevelInfo, c.level.Level())
	})
}

// a timer that fired while a newer change held the lock must not undo that change.
func TestServer_logLevelControl_revert_ignoresStaleTimer(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c, _ := newTestLogLevelControl(slog.LevelInfo)
		c.set(t.Context(), slog.LevelDebug, 15*time.Minute)
		stale := c.gen
		c.set(t.Context(), slog.LevelError, time.Hour)

		c.revert(t.Context(), stale)
		assert.Equal(t, slog.LevelError, c.level.Level())
	})
}

// newTestLogLevelControl returns a control whose logger filters on the controlled level, as in the exporter,
// and the log records it writes.
func newTestLogLevelControl(start slog.Level) (*logLevelControl, *logRecords) {
	level := new(slog.LevelVar)
	level.Set(start)
	logs := &logRecords{}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: level}))
	return newLogLevelControl(logger, level), logs
}

// logRecords collects JSON log lines; the revert timer writes from its own goroutine.
type logRecords struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logRecords) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// find returns the attributes of the first record with msg, failing the test when there is none.
func (l *logRecords) find(t *testing.T, msg string) map[string]string {
	t.Helper()

	l.mu.Lock()
	defer l.mu.Unlock()
	for line := range strings.SplitSeq(strings.TrimSpace(l.buf.String()), "\n") {
		var rec map[string]any
		if json.Unmarshal([]byte(line), &rec) != nil || rec["msg"] != msg {
			continue
		}
		out := map[string]string{}
		for k, v := range rec {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
		return out
	}
	t.Fatalf("no %q record in logs:\n%s", msg, l.buf.String())
	return nil
}

// request serves one request to h. An empty contentType sends none.
func request(t *testing.T, h http.Handler, method, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, logLevelPath, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeState(t *testing.T, rec *httptest.ResponseRecorder) logLevelState {
	t.Helper()

	var s logLevelState
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &s))
	return s
}
