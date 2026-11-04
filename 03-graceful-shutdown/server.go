package server

import (
	"context"
	"errors"
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
	serveErr chan error

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
	s.serveErr = make(chan error, 1)

	s.db = NewDBPool(s.workers, s.logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/job", s.handleJob)
	s.httpSrv = &http.Server{Handler: mux, ReadHeaderTimeout: s.requestTimeout}

	s.logger.Info("starting server", "addr", ln.Addr().String())

	for i := range s.workers {
		s.wg.Add(1)
		go s.worker(i)
	}
	go s.warmer()

	go func() {
		s.serveErr <- s.httpSrv.Serve(ln)
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("shutting down server")

	var errs []error

	err := s.httpSrv.Shutdown(ctx)
	if err != nil {
		errs = append(errs, err)
		s.logger.Warn("shutdown timeout", "error", err)
		err = s.httpSrv.Close()
		errs = append(errs, err)
	}

	err = <-s.serveErr
	if !errors.Is(err, http.ErrServerClosed) {
		errs = append(errs, err)
	}

	close(s.quit)
	s.wg.Wait()

	close(s.warmQuit)
	<-s.warmDone

	err = s.db.Close()
	if err != nil {
		errs = append(errs, err)
	}

	s.cancel()

	s.logger.Info("server stopped")

	return errors.Join(errs...)
}

func (s *Server) Addr() string {
	return s.ln.Addr().String()
}
