package aggregator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type Fetcher func(ctx context.Context, id int) (string, error)

type UserAggregator struct {
	profile Fetcher
	order   Fetcher
	timeout time.Duration
	logger  *slog.Logger
}

func (a *UserAggregator) Aggregate(ctx context.Context, id int) (string, error) {
	a.logger.Info("fetching", "id", id)

	var p, o string

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		res, err := a.profile(ctx, id)
		if err != nil {
			a.logger.Error("fetch profile failed", "err", err)
			return fmt.Errorf("fetch profile: %w", err)
		}
		p = res
		return nil
	})

	g.Go(func() error {
		res, err := a.order(ctx, id)
		if err != nil {
			a.logger.Error("fetch order failed", "err", err)
			return fmt.Errorf("fetch order: %w", err)
		}
		o = res
		return nil
	})

	if err := g.Wait(); err != nil {
		return "", err
	}

	p = strings.TrimPrefix(p, "Name: ")
	o = strings.TrimPrefix(o, "Orders: ")

	return fmt.Sprintf("User: %s | Orders: %s", p, o), nil
}

type Option func(*UserAggregator)

func WithLogger(logger *slog.Logger) Option {
	return func(a *UserAggregator) {
		a.logger = logger
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(a *UserAggregator) {
		a.timeout = timeout
	}
}

func New(profile, order Fetcher, opts ...Option) *UserAggregator {
	a := &UserAggregator{
		profile: profile,
		order:   order,
		timeout: 3 * time.Second,
		logger:  slog.Default(),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}
