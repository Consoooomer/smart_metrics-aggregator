package protocol

import (
	"encoding/binary"
	"io"
	"math"
	"time"
)

type Metric struct {
	Timestamp time.Time
	Name      string
	Value     float64
}

func ReadMetric(r io.Reader) (Metric, error) {
	var m Metric

	var buf8 [8]byte
	if _, err := io.ReadFull(r, buf8[:]); err != nil {
		return m, err
	}
	tsRaw := int64(binary.BigEndian.Uint64(buf8[:]))

	m.Timestamp = time.Unix(0, tsRaw)

	var nameLenBuf [1]byte
	if _, err := io.ReadFull(r, nameLenBuf[:]); err != nil {
		return m, err
	}
	nameLen := int(nameLenBuf[0])

	nameBytes := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBytes); err != nil {
		return m, err
	}
	m.Name = string(nameBytes)

	if _, err := io.ReadFull(r, buf8[:]); err != nil {
		return m, err
	}
	valBits := binary.BigEndian.Uint64(buf8[:])
	m.Value = math.Float64frombits(valBits)

	return m, nil
}
