package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"metrics-collector/internal/store"
)

type TimescaleClient struct {
	db     *sql.DB
	logger *log.Logger
}

func NewTimescaleClient(dsn string, logger *log.Logger) (*TimescaleClient, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	c := &TimescaleClient{
		db:     db,
		logger: logger,
	}

	if err := c.initSchema(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *TimescaleClient) initSchema() error {
	const ddl = `
CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS metrics (
    time TIMESTAMPTZ NOT NULL,
    name TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('metrics', 'time', if_not_exists => TRUE);
`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (c *TimescaleClient) InsertBatch(ctx context.Context, batch store.Batch) error {
	var (
		values []string
		args   []any
		argPos = 1
	)

	for name, points := range batch {
		for _, p := range points {
			values = append(values, fmt.Sprintf("($%d, $%d, $%d)", argPos, argPos+1, argPos+2))
			args = append(args, p.Timestamp, name, p.Value)
			argPos += 3
		}
	}

	if len(values) == 0 {
		return nil
	}

	query := "INSERT INTO metrics (time, name, value) VALUES " + strings.Join(values, ",")
	_, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}

func (c *TimescaleClient) Close() error {
	return c.db.Close()
}
