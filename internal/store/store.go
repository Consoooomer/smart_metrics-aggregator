package store

import (
	"context"
	_ "fmt"
	"log"
	_ "strings"
	"sync"
	"time"

	"metrics-collector/internal/protocol"
)

type Point struct {
	Timestamp time.Time
	Value     float64
}

type Batch map[string][]Point

type MetricStore struct {
	mu      sync.RWMutex
	metrics Batch
}

func NewMetricStore() *MetricStore {
	return &MetricStore{
		metrics: make(Batch),
	}
}

func (s *MetricStore) Add(m protocol.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	points := s.metrics[m.Name]
	points = append(points, Point{
		Timestamp: m.Timestamp,
		Value:     m.Value,
	})
	s.metrics[m.Name] = points
}

func (s *MetricStore) SnapshotAndClear() Batch {
	s.mu.Lock()
	defer s.mu.Unlock()

	old := s.metrics
	s.metrics = make(Batch)
	return old
}

type DBInserter interface {
	InsertBatch(ctx context.Context, batch Batch) error
}

func RunFlusher(ctx context.Context, s *MetricStore, db DBInserter, interval time.Duration, logger *log.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			batch := s.SnapshotAndClear()
			if len(batch) > 0 {
				if err := db.InsertBatch(context.Background(), batch); err != nil {
					logger.Printf("final flush error: %v", err)
				}
			}
			return
		case <-ticker.C:
			batch := s.SnapshotAndClear()
			if len(batch) == 0 {
				continue
			}
			if err := db.InsertBatch(ctx, batch); err != nil {
				logger.Printf("flush error: %v", err)
			}
		}
	}
}
