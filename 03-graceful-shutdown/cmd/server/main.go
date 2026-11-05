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

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	srv := server.New(server.WithLogger(log), server.WithWarmInterval(2*time.Second))

	err := srv.Start(ctx)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = srv.Stop(ctx)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
