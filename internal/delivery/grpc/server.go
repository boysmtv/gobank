package grpc

import (
	"context"
	"log/slog"
)

type Server struct {
	logger *slog.Logger
	addr   string
}

func NewServer(logger *slog.Logger, addr string) *Server {
	return &Server{
		logger: logger,
		addr:   addr,
	}
}

func (s *Server) Start(context.Context) error {
	s.logger.Info("grpc server scaffold initialized", "addr", s.addr)
	return nil
}
