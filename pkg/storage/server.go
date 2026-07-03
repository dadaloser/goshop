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
	ConnectToRedis(ctx, s.config)
	return nil
}

// Stop is a no-op because Start exits when the application context is cancelled.
func (s *Server) Stop(context.Context) error {
	return nil
}
