package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"metrics-collector/internal/config"
	"metrics-collector/internal/db"
	"metrics-collector/internal/server"
	"metrics-collector/internal/store"
)

func main() {
	logger := log.New(os.Stdout, "[metrics-collector] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	cfg, err := config.FromEnv()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}

	metricStore := store.NewMetricStore()

	tsClient, err := db.NewTimescaleClient(cfg.DBDSN, logger)
	if err != nil {
		logger.Fatalf("db init error: %v", err)
	}
	defer tsClient.Close()

	srv := server.New(cfg.ListenAddr, metricStore, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go store.RunFlusher(ctx, metricStore, tsClient, cfg.FlushInterval, logger)

	if err := srv.Run(ctx); err != nil {
		logger.Fatalf("server error: %v", err)
	}

	logger.Printf("shutdown complete")
}
