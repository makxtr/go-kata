package server

import (
	"fmt"
	"time"
)

type Job struct {
	ID       int
	Result   chan string
	Duration time.Duration
}

func newJob(id int, d time.Duration) Job {
	return Job{ID: id, Duration: d, Result: make(chan string, 1)}
}

func (s *Server) worker(id int) {
	defer func() {
		s.logger.Debug("worker stopped", "id", id)
		s.wg.Done()
	}()

	s.logger.Debug("worker started", "id", id)

	for {
		select {
		case job := <-s.jobs:

			select {
			case <-time.After(job.Duration):
			case <-s.quit:
				return
			}

			job.Result <- fmt.Sprintf("job %d done by worker %d", job.ID, id)

		case <-s.quit:
			return
		}
	}
}
