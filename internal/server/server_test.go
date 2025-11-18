package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"math"
	"metrics-collector/internal/store"
	"net"
	"testing"
	"time"
)

func buildMessage(ts time.Time, name string, value float64) []byte {
	buf := &bytes.Buffer{}
	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, uint64(ts.UnixNano()))
	buf.Write(b8)

	if len(name) > 255 {
		panic("name too long")
	}
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)

	binary.BigEndian.PutUint64(b8, math.Float64bits(value))
	buf.Write(b8)
	return buf.Bytes()
}

func TestServer_ReceivesAndStoresMetric(t *testing.T) {
	ms := store.NewMetricStore()
	logger := log.New(io.Discard, "", 0)

	srv := New("127.0.0.1:0", ms, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open listener: %v", err)
	}
	defer l.Close()

	go func() {
		if err := srv.RunOnListener(ctx, l); err != nil {
			t.Logf("server exited with error: %v", err)
		}
	}()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer conn.Close()

	ts := time.Unix(10, 20)
	name := "test_metric"
	value := 1.5
	msg := buildMessage(ts, name, value)

	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(20 * time.Millisecond)

	batch := ms.SnapshotAndClear()
	points, ok := batch[name]
	if !ok {
		t.Fatalf("expected metric %q in store, got none", name)
	}
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
