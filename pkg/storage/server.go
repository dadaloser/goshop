package storage

import (
	"context"
	"errors"
)

// Server runs the Redis connection loop under the application lifecycle.
type Server struct {
	config *Config
}

// NewServer creates a lifecycle-managed Redis server.
func NewServer(config *Config) *Server {
	return &Server{config: config}
}

// Start connects to Redis and retries until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	if s.config == nil {
		return errors.New("redis config is required")
	}
	if err := s.config.Validate(); err != nil {
		return err
	}
	ConnectToRedis(ctx, s.config)
	return nil
}

// Stop closes pools after the application context has cancelled the probe loop.
func (s *Server) Stop(context.Context) error {
	return CloseRedis()
}
