package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, opts ...Option) *Server {
	t.Helper()

	def := []Option{
		WithAddr(":0"),
		WithWarmInterval(10 * time.Millisecond),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	}

	srv := New(append(def, opts...)...)
	if err := srv.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	return srv
}

func waitGoroutines(t *testing.T, want int) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("goroutines after Stop = %d; want %d", runtime.NumGoroutine(), want)
}

func TestServer_HappyPath(t *testing.T) {
	before := runtime.NumGoroutine()

	srv := newTestServer(t)

	resp, err := http.Get("http://" + srv.Addr() + "/job")
	if err != nil {
		t.Fatalf("GET /job: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /job status = %d; want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "done by worker") {
		t.Fatalf("body = %q; want to contain %q", body, "done by worker")
	}

	http.DefaultTransport.(*http.Transport).CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	waitGoroutines(t, before)
}
