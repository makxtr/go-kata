package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

func TestServer_SuddenDeath(t *testing.T) {
	var buf bytes.Buffer
	before := runtime.NumGoroutine()

	srv := newTestServer(t, WithLogger(slog.New(slog.NewTextHandler(&buf, nil))))

	var wg sync.WaitGroup
	var ok, failed atomic.Int64

	for range 100 {
		wg.Go(func() {
			resp, err := http.Get("http://" + srv.Addr() + "/job")
			if err != nil {
				failed.Add(1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				failed.Add(1)
				return
			}

			_, err = io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
				return
			}
			ok.Add(1)
		})
	}
	time.Sleep(5 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	wg.Wait()
	http.DefaultTransport.(*http.Transport).CloseIdleConnections()

	if ok.Load() == 0 {
		t.Fatalf("ok = %d, failed = %d; want ok > 0", ok.Load(), failed.Load())
	}
	if !strings.Contains(buf.String(), "shutting down") {
		t.Fatalf("buf = %q; want to contain %q", buf.String(), "shutting down")
	}

	t.Logf("ok = %d, failed = %d", ok.Load(), failed.Load())

	waitGoroutines(t, before)
}
