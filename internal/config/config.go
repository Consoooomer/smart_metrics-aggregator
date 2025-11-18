package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	ListenAddr    string
	DBDSN         string
	FlushInterval time.Duration
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func FromEnv() (Config, error) {
	cfg := Config{
		ListenAddr:    getEnv("LISTEN_ADDR", ":9000"),
		DBDSN:         os.Getenv("DB_DSN"),
		FlushInterval: 5 * time.Second,
	}

	if fi := os.Getenv("FLUSH_INTERVAL"); fi != "" {
		d, err := time.ParseDuration(fi)
		if err != nil {
			return Config{}, fmt.Errorf("invalid FLUSH_INTERVAL: %w", err)
		}
		cfg.FlushInterval = d
	}

	if cfg.DBDSN == "" {
		return Config{}, fmt.Errorf("DB_DSN is required")
	}

	return cfg, nil
}
