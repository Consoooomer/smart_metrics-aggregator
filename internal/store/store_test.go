package store

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"metrics-collector/internal/protocol"
)

type dummyDB struct {
	mu      sync.Mutex
	batches []Batch
}

func (d *dummyDB) InsertBatch(_ context.Context, b Batch) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	copyBatch := make(Batch, len(b))
	for k, v := range b {
		pointsCopy := make([]Point, len(v))
		copy(pointsCopy, v)
		copyBatch[k] = pointsCopy
	}
	d.batches = append(d.batches, copyBatch)
	return nil
}

func (d *dummyDB) totalPoints() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	total := 0
	for _, b := range d.batches {
		for _, points := range b {
			total += len(points)
		}
	}
	return total
}

func TestMetricStore_AddAndSnapshot(t *testing.T) {
	s := NewMetricStore()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s.Add(protocol.Metric{
					Timestamp: time.Now(),
					Name:      "cpu_load",
					Value:     float64(id*100 + j),
				})
			}
		}(i)
	}
	wg.Wait()

	batch := s.SnapshotAndClear()
	if len(batch) != 1 {
		t.Fatalf("expected 1 metric name, got %d", len(batch))
	}
	points := batch["cpu_load"]
	if len(points) != 1000 {
		t.Fatalf("expected 1000 points, got %d", len(points))
	}

	batch2 := s.SnapshotAndClear()
	if len(batch2) != 0 {
		t.Fatalf("expected empty batch after clear, got %d", len(batch2))
	}
}

func TestRunFlusher_FlushesPeriodically(t *testing.T) {
	s := NewMetricStore()
	db := &dummyDB{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interval := 10 * time.Millisecond
	go RunFlusher(ctx, s, db, interval, nil)

	const total = 50
	for i := 0; i < total; i++ {
		s.Add(protocol.Metric{
			Timestamp: time.Now(),
			Name:      "memory_usage",
			Value:     float64(i),
		})
	}

	time.Sleep(3 * interval)

	cancel()
	time.Sleep(interval)

	got := db.totalPoints()
	if got != total {
		t.Fatalf("expected %d points flushed, got %d", total, got)
	}
}

func TestPointStructure(t *testing.T) {
	ts := time.Now()
	value := 3.14

	s := NewMetricStore()
	s.Add(protocol.Metric{
		Timestamp: ts,
		Name:      "temp",
		Value:     value,
	})

	batch := s.SnapshotAndClear()
	points := batch["temp"]
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}

	if !points[0].Timestamp.Equal(ts) {
		t.Fatalf("timestamp mismatch: expected %v, got %v", ts, points[0].Timestamp)
	}
	if math.Abs(points[0].Value-value) > 1e-9 {
		t.Fatalf("value mismatch: expected %v, got %v", value, points[0].Value)
	}
}
