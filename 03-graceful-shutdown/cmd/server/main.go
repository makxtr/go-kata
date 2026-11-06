package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	srv := server.New(server.WithLogger(logger), server.WithWarmInterval(2*time.Second))

	if err := srv.Start(ctx); err != nil {
		logger.Error("start failed", "err", err)
		os.Exit(1)
	}

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error("stop failed", "err", err)
		os.Exit(1)
	}
}
