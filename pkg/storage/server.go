package storage

import (
	"context"
)

// Server runs the Redis connection loop under the application lifecycle.
type Server struct {
	client *Client
}

// NewServer creates a lifecycle-managed Redis server.
func NewServer(config *Config) (*Server, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Server{client: client}, nil
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
	return s.client.Start(ctx)
}

// Stop closes pools after the application context has cancelled the probe loop.
func (s *Server) Stop(context.Context) error {
	if s == nil {
		return nil
	}
	return s.client.Close()
}
