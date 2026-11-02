package server

import (
	"log/slog"
	"sync"
	"time"
)

type Server struct {
	addr           string
	requestTimeout time.Duration
	workers        int
	warmInterval   time.Duration
	logger         *slog.Logger
	jobs           chan Job
	quit           chan struct{}
	wg             sync.WaitGroup
}

type Option func(*Server)

func WithAddr(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.requestTimeout = timeout
	}
}

func WithWorkers(workers int) Option {
	return func(s *Server) {
		s.workers = workers
	}
}

func WithWarmInterval(warmup time.Duration) Option {
	return func(s *Server) {
		s.warmInterval = warmup
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

func New(opts ...Option) *Server {
	s := &Server{
		addr:           ":8080",
		requestTimeout: 5 * time.Second,
		workers:        5,
		warmInterval:   30 * time.Second,
		logger:         slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
