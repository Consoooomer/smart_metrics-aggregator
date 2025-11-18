package server

import (
	"context"
	"log"
	"net"

	"metrics-collector/internal/protocol"
	"metrics-collector/internal/store"
)

type Server struct {
	addr   string
	store  *store.MetricStore
	logger *log.Logger
}

func New(addr string, s *store.MetricStore, logger *log.Logger) *Server {
	return &Server{
		addr:   addr,
		store:  s,
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	return s.run(ctx, l)
}

func (s *Server) run(ctx context.Context, l net.Listener) error {
	s.logger.Printf("listening on %s", l.Addr())

	go func() {
		<-ctx.Done()
		if s.logger != nil {
			s.logger.Printf("shutting down listener...")
		}
		_ = l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			if s.logger != nil {
				s.logger.Printf("accept error: %v", err)
			}
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *Server) RunOnListener(ctx context.Context, l net.Listener) error {
	return s.run(ctx, l)
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	s.logger.Printf("new connection from %s", conn.RemoteAddr())

	for {
		m, err := protocol.ReadMetric(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				s.logger.Printf("read timeout from %s: %v", conn.RemoteAddr(), err)
				return
			}

			s.logger.Printf("read error from %s: %v", conn.RemoteAddr(), err)
			return
		}
		s.store.Add(m)
	}
}
