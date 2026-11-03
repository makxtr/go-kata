package server

import (
	"context"
	"fmt"
	"net/http"
)

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	job := Job{ID: int(s.nextID.Add(1)), Result: make(chan string, 1)}

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
