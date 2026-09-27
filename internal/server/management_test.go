package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	ready := new(atomic.Bool)
	h := readyHandler(slog.New(slog.DiscardHandler), ready)

	rec := serve(t, h, http.MethodGet, "")
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	ready.Store(true)
	rec = serve(t, h, http.MethodGet, "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLogLevelGet(t *testing.T) {
	t.Parallel()

	c := newTestLogLevelControl()
	rec := serve(t, c, http.MethodGet, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"level":"info"}`, rec.Body.String())
}

func TestLogLevelChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		method    string
		body      string
		wantCode  int
		wantLevel slog.Level
	}{
		{name: "put", method: http.MethodPut, body: `{"level":"debug"}`, wantCode: http.StatusOK, wantLevel: slog.LevelDebug},
		{name: "post", method: http.MethodPost, body: `{"level":"warn"}`, wantCode: http.StatusOK, wantLevel: slog.LevelWarn},
		{name: "upper case", method: http.MethodPut, body: `{"level":"ERROR"}`, wantCode: http.StatusOK, wantLevel: slog.LevelError},
		{name: "unknown level", method: http.MethodPut, body: `{"level":"trace"}`, wantCode: http.StatusBadRequest},
		{name: "slog offset level", method: http.MethodPut, body: `{"level":"info+2"}`, wantCode: http.StatusBadRequest},
		{name: "missing level", method: http.MethodPut, body: `{}`, wantCode: http.StatusBadRequest},
		{name: "unknown field", method: http.MethodPut, body: `{"level":"debug","x":1}`, wantCode: http.StatusBadRequest},
		{name: "not json", method: http.MethodPut, body: `debug`, wantCode: http.StatusBadRequest},
		{name: "invalid for", method: http.MethodPut, body: `{"level":"debug","for":"soon"}`, wantCode: http.StatusBadRequest},
		{name: "zero for", method: http.MethodPut, body: `{"level":"debug","for":"0s"}`, wantCode: http.StatusBadRequest},
		{name: "negative for", method: http.MethodPut, body: `{"level":"debug","for":"-1m"}`, wantCode: http.StatusBadRequest},
		{name: "for too long", method: http.MethodPut, body: `{"level":"debug","for":"25h"}`, wantCode: http.StatusBadRequest},
		{
			name: "body too large", method: http.MethodPut,
			body:     `{"level":"debug","for":"` + strings.Repeat("1", maxLogLevelBody) + `s"}`,
			wantCode: http.StatusBadRequest,
		},
		{name: "delete", method: http.MethodDelete, wantCode: http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestLogLevelControl()
			rec := serve(t, c, tt.method, tt.body)
			assert.Equal(t, tt.wantCode, rec.Code, rec.Body.String())

			want := slog.LevelInfo
			if tt.wantCode == http.StatusOK {
				want = tt.wantLevel
			}
			assert.Equal(t, want, c.level.Level())
		})
	}
}

func TestLogLevelMethodNotAllowedHeader(t *testing.T) {
	t.Parallel()

	rec := serve(t, newTestLogLevelControl(), http.MethodDelete, "")
	assert.Equal(t, "GET, HEAD, PUT, POST", rec.Header().Get("Allow"))
}

func TestLogLevelTemporaryReverts(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c := newTestLogLevelControl()

		rec := serve(t, c, http.MethodPut, `{"level":"debug","for":"15m"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		state := decodeState(t, rec)
		assert.Equal(t, "debug", state.Level)
		assert.Equal(t, "info", state.RevertTo)
		require.NotNil(t, state.RevertAt)
		assert.Equal(t, time.Now().Add(15*time.Minute).UTC(), *state.RevertAt)

		time.Sleep(15*time.Minute - time.Second)
		synctest.Wait()
		assert.Equal(t, slog.LevelDebug, c.level.Level())

		time.Sleep(time.Second)
		synctest.Wait()
		assert.Equal(t, slog.LevelInfo, c.level.Level())
		assert.JSONEq(t, `{"level":"info"}`, serve(t, c, http.MethodGet, "").Body.String())
	})
}

func TestLogLevelPermanentCancelsRevert(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c := newTestLogLevelControl()

		require.Equal(t, http.StatusOK, serve(t, c, http.MethodPut, `{"level":"debug","for":"15m"}`).Code)
		require.Equal(t, http.StatusOK, serve(t, c, http.MethodPut, `{"level":"warn"}`).Code)

		time.Sleep(time.Hour)
		synctest.Wait()
		assert.Equal(t, slog.LevelWarn, c.level.Level())
	})
}

func TestLogLevelTemporaryReplacesTemporary(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		c := newTestLogLevelControl()

		require.Equal(t, http.StatusOK, serve(t, c, http.MethodPut, `{"level":"debug","for":"15m"}`).Code)
		rec := serve(t, c, http.MethodPut, `{"level":"error","for":"1h"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		// the second change still reverts to the permanent level, not to the first temporary one
		assert.Equal(t, "info", decodeState(t, rec).RevertTo)

		time.Sleep(30 * time.Minute)
		synctest.Wait()
		assert.Equal(t, slog.LevelError, c.level.Level(), "the first change's timer must not revert")

		time.Sleep(30 * time.Minute)
		synctest.Wait()
		assert.Equal(t, slog.LevelInfo, c.level.Level())
	})
}

// newTestLogLevelControl returns a control whose level starts at info, the slog.LevelVar zero value.
func newTestLogLevelControl() *logLevelControl {
	return newLogLevelControl(slog.New(slog.DiscardHandler), new(slog.LevelVar))
}

func serve(t *testing.T, h http.Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, logLevelPath, strings.NewReader(body))
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
