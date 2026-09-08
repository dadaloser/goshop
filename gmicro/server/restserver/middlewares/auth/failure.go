package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	// ErrMissingCredentials indicates that no authentication credentials were supplied.
	ErrMissingCredentials = errors.New("authentication credentials are missing")
	// ErrInvalidAuthorization indicates that the Authorization header is malformed.
	ErrInvalidAuthorization = errors.New("authorization header is invalid")
	// ErrInvalidCredentials indicates that supplied credentials could not be verified.
	ErrInvalidCredentials = errors.New("authentication credentials are invalid")
	// ErrInvalidToken indicates that the bearer token could not be verified.
	ErrInvalidToken = errors.New("authentication token is invalid")
	// ErrExpiredCredentials indicates that otherwise valid credentials have expired.
	ErrExpiredCredentials = errors.New("authentication credentials have expired")
	// ErrUnauthorized indicates that the configured authorization callback rejected the identity.
	ErrUnauthorized = errors.New("authentication identity is unauthorized")
)

// FailureResponder writes an HTTP response for an authentication failure.
// Applications inject this adapter to select their own public error protocol.
type FailureResponder func(*gin.Context, error)

// Option configures an authentication strategy.
type Option func(*strategyOptions)

type strategyOptions struct {
	failureResponder FailureResponder
}

// WithFailureResponder configures the adapter used to render authentication failures.
func WithFailureResponder(responder FailureResponder) Option {
	return func(options *strategyOptions) {
		options.failureResponder = responder
	}
}

func resolveFailureResponder(options []Option) FailureResponder {
	config := strategyOptions{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config.failureResponder
}

func reject(c *gin.Context, responder FailureResponder, err error) {
	if responder != nil {
		responder(c, err)
	} else {
		c.Status(http.StatusUnauthorized)
	}
	c.Abort()
}
