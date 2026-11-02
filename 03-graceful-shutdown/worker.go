package server

import (
	"fmt"
	"time"
)

type Job struct {
	ID     int
	Result chan string
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
			time.Sleep(10 * time.Millisecond)
			select {
			case job.Result <- fmt.Sprintf("job %d done by worker %d", job.ID, id):
			case <-s.quit:
				return
			}

		case <-s.quit:
			return
		}
	}
}
