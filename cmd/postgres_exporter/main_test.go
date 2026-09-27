package main

import (
	"context"
	"io"
	"testing"

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
}

// --web.enabled-pprof was renamed before any release; the old name is not an alias.
func TestMain_parseFlags_pprofUsesOnlyTheNewName(t *testing.T) {
	t.Parallel()

	cfg, _, err := parseFlags([]string{"--web.enable-pprof"}, io.Discard)
	require.NoError(t, err)
	assert.True(t, cfg.Pprof)

	_, _, err = parseFlags([]string{"--web.enabled-pprof"}, io.Discard)
	require.Error(t, err)
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

func TestMain_run_rejectsInvalidFlags(t *testing.T) {
	t.Parallel()

	err := run(t.Context(), []string{"--log.level=trace"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "parse command line")
}

// a canceled context is a clean shutdown, not an error.
func TestMain_run_returnsNilOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := run(ctx, []string{
		"--web.listen-address=127.0.0.1:0", "--db.data-source=host=127.0.0.1 port=1",
	}, io.Discard, io.Discard)
	require.NoError(t, err)
}
