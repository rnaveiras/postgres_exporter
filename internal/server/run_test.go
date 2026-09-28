package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig binds a free port and points the data source at a closed one: these tests exercise the HTTP
// server, never Postgres.
func testConfig() Config {
	return Config{
		ListenAddress: "127.0.0.1:0",
		MetricsPath:   "/metrics",
		DataSource:    "host=127.0.0.1 port=1 connect_timeout=1",
	}
}

// a busy address must stop the process instead of leaving one that serves nothing.
func TestServer_Run_failsWhenAddressInUse(t *testing.T) {
	t.Parallel()

	busy, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })

	cfg := testConfig()
	cfg.ListenAddress = busy.Addr().String()
	err = Run(t.Context(), cfg, slog.New(slog.DiscardHandler), new(slog.LevelVar))
	require.ErrorContains(t, err, "listen")
}

// a canceled context is a clean shutdown, not an error.
func TestServer_Run_returnsNilOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.NoError(t, Run(ctx, testConfig(), slog.New(slog.DiscardHandler), new(slog.LevelVar)))
}

func TestServer_server_serve_answersProbesUntilCanceled(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.serve(ctx) }()

	for _, path := range []string{healthyPath, readyPath} {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.Equal(c, http.StatusOK, get(c, "http://"+s.addr()+path))
		}, 5*time.Second, 10*time.Millisecond, path)
	}

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after cancel")
	}
}

// a server that stops accepting connections must end the process, not leave it running and unreachable.
func TestServer_server_serve_returnsServeError(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	require.NoError(t, s.listener.Close())
	require.Error(t, s.serve(t.Context()))
}

func newTestServer(t *testing.T) *server {
	t.Helper()

	s, err := newServer(t.Context(), testConfig(), slog.New(slog.DiscardHandler), new(slog.LevelVar))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.listener.Close() })
	return s
}

// probeClient opens a connection per request. A pooled client can leave a spare connection that never sends
// a request, and Shutdown waits up to 5s for such a connection, which would make the cancel check flaky.
var probeClient = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

// get returns the status code of a GET to url, or 0 when the request fails.
func get(c *assert.CollectT, url string) int {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	require.NoError(c, err)
	resp, err := probeClient.Do(req)
	if err != nil {
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}
