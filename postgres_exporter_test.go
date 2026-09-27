package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	pgx "github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlagsDefaults(t *testing.T) {
	t.Parallel()

	cfg, _, err := parseFlags(nil, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, []string{"cloudsqladmin", "rdsadmin", "azure_maintenance", "azure_sys"}, cfg.ExcludedDatabases)
	assert.False(t, cfg.Pprof)
	assert.True(t, cfg.LegacyNames)
	assert.Empty(t, cfg.deprecated)
}

func TestParseFlagsDeprecatedPprof(t *testing.T) {
	t.Parallel()

	cfg, _, err := parseFlags([]string{"--web.enabled-pprof"}, io.Discard)
	require.NoError(t, err)
	assert.True(t, cfg.Pprof)
	assert.Equal(t, []string{"--web.enabled-pprof -> --web.enable-pprof"}, cfg.deprecated)

	cfg, _, err = parseFlags([]string{"--web.enable-pprof"}, io.Discard)
	require.NoError(t, err)
	assert.True(t, cfg.Pprof)
	assert.Empty(t, cfg.deprecated)

	// an explicit --no- form still uses the old name, so it must warn before the alias is removed.
	for _, arg := range []string{"--no-web.enabled-pprof"} {
		cfg, _, err = parseFlags([]string{arg}, io.Discard)
		require.NoError(t, err, arg)
		assert.False(t, cfg.Pprof, arg)
		assert.Equal(t, []string{"--web.enabled-pprof -> --web.enable-pprof"}, cfg.deprecated, arg)
	}
}

func TestMuxRoutes(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	connConfig, err := pgx.ParseConfig("host=127.0.0.1 port=1 user=nobody connect_timeout=1")
	require.NoError(t, err)

	tests := []struct {
		name     string
		pprof    bool
		adminAPI bool
		method   string
		path     string
		want     int
	}{
		{name: "pprof cmdline enabled", pprof: true, path: "/debug/pprof/cmdline", want: http.StatusOK},
		{name: "pprof index enabled", pprof: true, path: "/debug/pprof/", want: http.StatusOK},
		{name: "pprof named profile enabled", pprof: true, path: "/debug/pprof/heap", want: http.StatusOK},
		{name: "pprof disabled", path: "/debug/pprof/cmdline", want: http.StatusNotFound},
		{name: "loglevel endpoint removed", path: "/admin/loglevel", want: http.StatusNotFound},
		{name: "landing page", path: "/", want: http.StatusOK},
		{name: "healthy", path: "/-/healthy", want: http.StatusOK},
		{name: "healthy head", method: http.MethodHead, path: "/-/healthy", want: http.StatusOK},
		{name: "ready", path: "/-/ready", want: http.StatusOK},
		{name: "ready head", method: http.MethodHead, path: "/-/ready", want: http.StatusOK},
		{name: "log level without admin api", path: "/-/log-level", want: http.StatusNotFound},
		{
			name: "log level change without admin api", method: http.MethodPut, path: "/-/log-level",
			want: http.StatusNotFound,
		},
		{name: "log level with admin api", adminAPI: true, path: "/-/log-level", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ready := new(atomic.Bool)
			ready.Store(true)
			cfg := flagConfig{MetricsPath: "/metrics", Pprof: tt.pprof, AdminAPI: tt.adminAPI}
			mux := newMux(logger, connConfig, cfg, new(slog.LevelVar), ready)

			method := tt.method
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequestWithContext(t.Context(), method, tt.path, http.NoBody)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestParseFlagsAdminAPI(t *testing.T) {
	t.Parallel()

	cfg, _, err := parseFlags(nil, io.Discard)
	require.NoError(t, err)
	assert.False(t, cfg.AdminAPI)

	cfg, _, err = parseFlags([]string{"--web.enable-admin-api"}, io.Discard)
	require.NoError(t, err)
	assert.True(t, cfg.AdminAPI)
}
