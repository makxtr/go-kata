package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func parseSleep(raw string) (time.Duration, error) {
	if raw == "" {
		return 10 * time.Millisecond, nil
	}

	return time.ParseDuration(raw)
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	d, err := parseSleep(r.URL.Query().Get("sleep"))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid sleep parameter: %v", err), http.StatusBadRequest)
		return
	}

	job := newJob(int(s.nextID.Add(1)), d)

	select {
	case s.jobs <- job:
	case <-ctx.Done():
		http.Error(w, "server busy", http.StatusServiceUnavailable)
		return
	}

	select {
	case res := <-job.Result:
		fmt.Fprintln(w, res)
	case <-ctx.Done():
		http.Error(w, "request timeout", http.StatusGatewayTimeout)
	}
}
