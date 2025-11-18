package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func buildMessage(ts time.Time, name string, value float64) []byte {
	buf := &bytes.Buffer{}
	b8 := make([]byte, 8)

	binary.BigEndian.PutUint64(b8, uint64(ts.UnixNano()))
	buf.Write(b8)

	if len(name) > 255 {
		log.Fatalf("metric name too long: %d", len(name))
	}
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)

	binary.BigEndian.PutUint64(b8, math.Float64bits(value))
	buf.Write(b8)

	return buf.Bytes()
}

func main() {
	var (
		addr        string
		total       int
		concurrency int
		name        string
	)
	flag.StringVar(&addr, "addr", "127.0.0.1:9000", "TCP address of metrics collector")
	flag.IntVar(&total, "metrics", 100_000, "total number of metrics to send")
	flag.IntVar(&concurrency, "concurrency", 4, "number of concurrent connections")
	flag.StringVar(&name, "name", "cpu_load", "metric name")
	flag.Parse()

	if total <= 0 || concurrency <= 0 {
		log.Fatalf("metrics and concurrency must be > 0")
	}

	fmt.Printf("loadgen: addr=%s, total=%d, concurrency=%d, name=%s\n",
		addr, total, concurrency, name)

	start := time.Now()

	var sent int64
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Printf("[worker %d] dial error: %v", id, err)
				return
			}
			defer conn.Close()

			for {
				n := int(atomic.AddInt64(&sent, 1))
				if n > total {
					return
				}

				ts := time.Now()
				value := float64(n%100) + float64(id)
				msg := buildMessage(ts, name, value)

				if _, err := conn.Write(msg); err != nil {
					log.Printf("[worker %d] write error: %v", id, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)
	tps := float64(total) / elapsed.Seconds()

	fmt.Printf("done: sent=%d, elapsed=%s, throughput=%.2f metrics/sec\n",
		total, elapsed, tps)
}
