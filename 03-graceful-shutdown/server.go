package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	// settings
	addr           string
	requestTimeout time.Duration
	warmInterval   time.Duration
	workers        int
	logger         *slog.Logger

	// lifecycle
	jobs     chan Job
	quit     chan struct{}
	wg       sync.WaitGroup
	nextID   atomic.Int64
	warmQuit chan struct{}
	warmDone chan struct{}

	// resources
	db      *DBPool
	httpSrv *http.Server
	ln      net.Listener
	ctx     context.Context
	cancel  context.CancelFunc
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

func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("could not listen on %s: %w", s.addr, err)
	}
	s.ln = ln

	s.ctx, s.cancel = context.WithCancel(ctx)

	s.jobs = make(chan Job)
	s.quit = make(chan struct{})
	s.warmQuit = make(chan struct{})
	s.warmDone = make(chan struct{})

	s.db = NewDBPool(s.workers, s.logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/job", s.handleJob)
	s.httpSrv = &http.Server{Handler: mux, ReadHeaderTimeout: s.requestTimeout}

	s.logger.Info("starting server", "addr", ln.Addr().String())

	return nil
}

func (s *Server) Addr() string {
	return s.ln.Addr().String()
}
