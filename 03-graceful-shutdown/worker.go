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
	defer s.wg.Done()

	for {
		select {
		case job := <-s.jobs:
			time.Sleep(10 * time.Millisecond)
			job.Result <- fmt.Sprintf("job %d done by worker %d", job.ID, id)

		case <-s.quit:
			return
		}
	}
}
