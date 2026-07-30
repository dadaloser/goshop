// Package eventbus provides the shared reliable-message transport boundary.
// Business services must write an outbox record before calling Publisher.Publish.
package eventbus

import (
	"context"
	stdErrors "errors"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const DefaultStream = "GOSHOP_EVENTS"

// Config describes a JetStream connection. Credentials are intentionally
// supplied through the NATS URL or NATS client options, never event payloads.
type Config struct {
	URL            string
	Stream         string
	ConnectTimeout time.Duration
}

func (c Config) normalized() Config {
	c.URL = strings.TrimSpace(c.URL)
	c.Stream = strings.TrimSpace(c.Stream)
	if c.Stream == "" {
		c.Stream = DefaultStream
	}
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	return c
}

func (c Config) Validate() error {
	return c.validate()
}

func (c Config) validate() error {
	c = c.normalized()
	if c.URL == "" {
		return ErrURLRequired
	}
	return nil
}

var ErrURLRequired = stdErrors.New("nats url is required")

// Event is the common envelope stored in an outbox and published to JetStream.
// ID is used as the NATS de-duplication key and must remain stable on retries.
type Event struct {
	ID            string
	Subject       string
	OccurredAt    time.Time
	Payload       []byte
	CorrelationID string
}

func (e Event) validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return ErrEventIDRequired
	}
	if strings.TrimSpace(e.Subject) == "" {
		return ErrSubjectRequired
	}
	if len(e.Payload) == 0 {
		return ErrPayloadRequired
	}
	return nil
}

var (
	ErrEventIDRequired = stdErrors.New("event id is required")
	ErrSubjectRequired = stdErrors.New("event subject is required")
	ErrPayloadRequired = stdErrors.New("event payload is required")
)

// Publisher publishes an already-persisted outbox event. A successful return
// only means JetStream accepted the message; callers must then mark the outbox
// record delivered in their own database.
type Publisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func Connect(cfg Config) (*Publisher, error) {
	cfg = cfg.normalized()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	nc, err := nats.Connect(cfg.URL, nats.Timeout(cfg.ConnectTimeout))
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &Publisher{nc: nc, js: js}, nil
}

func (p *Publisher) Close() {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
}

func (p *Publisher) Publish(ctx context.Context, event Event) error {
	if p == nil || p.js == nil {
		return ErrPublisherUnavailable
	}
	if err := event.validate(); err != nil {
		return err
	}
	msg := nats.NewMsg(event.Subject)
	msg.Data = append([]byte(nil), event.Payload...)
	msg.Header.Set(nats.MsgIdHdr, event.ID)
	if event.CorrelationID != "" {
		msg.Header.Set("X-Correlation-ID", event.CorrelationID)
	}
	if !event.OccurredAt.IsZero() {
		msg.Header.Set("X-Occurred-At", event.OccurredAt.UTC().Format(time.RFC3339Nano))
	}
	_, err := p.js.PublishMsg(msg, nats.Context(ctx))
	return err
}

var ErrPublisherUnavailable = stdErrors.New("event publisher is unavailable")
