package protocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
	"time"
)

func buildMessage(ts time.Time, name string, value float64) []byte {
	buf := &bytes.Buffer{}
	b8 := make([]byte, 8)

	binary.BigEndian.PutUint64(b8, uint64(ts.UnixNano()))
	buf.Write(b8)

	if len(name) > 255 {
		panic("name too long for test")
	}
	buf.WriteByte(byte(len(name)))

	buf.WriteString(name)

	binary.BigEndian.PutUint64(b8, math.Float64bits(value))
	buf.Write(b8)

	return buf.Bytes()
}

func TestReadMetric_OK(t *testing.T) {
	ts := time.Unix(123, 456)
	name := "cpu_load"
	value := 42.5

	msg := buildMessage(ts, name, value)
	m, err := ReadMetric(bytes.NewReader(msg))
	if err != nil {
		t.Fatalf("ReadMetric returned error: %v", err)
	}

	if !m.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, m.Timestamp)
	}
	if m.Name != name {
		t.Fatalf("expected name %q, got %q", name, m.Name)
	}
	if math.Abs(m.Value-value) > 1e-9 {
		t.Fatalf("expected value %v, got %v", value, m.Value)
	}
}

func TestReadMetric_InvalidNameLength(t *testing.T) {
	// timestamp + nameLen=0
	buf := &bytes.Buffer{}
	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, uint64(time.Now().UnixNano()))
	buf.Write(b8)
	buf.WriteByte(0)

	_, err := ReadMetric(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatalf("expected error for invalid name length, got nil")
	}
}

func TestReadMetric_Truncated(t *testing.T) {
	ts := time.Now()
	name := "test"
	value := 1.23

	full := buildMessage(ts, name, value)
	truncated := full[:len(full)-3]

	_, err := ReadMetric(bytes.NewReader(truncated))
	if err == nil {
		t.Fatalf("expected error for truncated message, got nil")
	}
	if err != io.ErrUnexpectedEOF && err != io.EOF {
		t.Logf("got error type: %T, value: %v", err, err)
	}
}
