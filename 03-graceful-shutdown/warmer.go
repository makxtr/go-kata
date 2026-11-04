package server

import "time"

func (s *Server) warmer() {
	defer close(s.warmDone)

	ticker := time.NewTicker(s.warmInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.logger.Info("cache warmed")
		case <-s.warmQuit:
			return
		}
	}
}
