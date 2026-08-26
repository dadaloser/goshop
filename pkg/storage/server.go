package storage

import (
	"context"
)

// Server runs the Redis connection loop under the application lifecycle.
type Server struct {
	client  *Client
	initErr error
}

// NewServer creates a lifecycle-managed Redis server.
func NewServer(config *Config) *Server {
	client, err := NewClient(config)
	return &Server{client: client, initErr: err}
}

// Client returns the Client owned by this lifecycle server.
func (s *Server) Client() *Client {
	if s == nil {
		return nil
	}
	return s.client
}

// Start connects to Redis and retries until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	if s == nil {
		return ErrRedisIsDown
	}
	if s.initErr != nil {
		return s.initErr
	}
	return s.client.Start(ctx)
}

// Stop closes pools after the application context has cancelled the probe loop.
func (s *Server) Stop(context.Context) error {
	if s == nil {
		return nil
	}
	return s.client.Close()
}
