package eventbus

import (
	"context"
	"errors"
	"testing"
)

func TestConfigRequiresNATSURL(t *testing.T) {
	if err := (Config{}).Validate(); !errors.Is(err, ErrURLRequired) {
		t.Fatalf("Validate() error = %v, want ErrURLRequired", err)
	}
}

func TestPublisherRejectsInvalidEventsBeforeNetwork(t *testing.T) {
	publisher := &Publisher{}
	if err := publisher.Publish(context.Background(), Event{}); !errors.Is(err, ErrPublisherUnavailable) {
		t.Fatalf("Publish() error = %v, want ErrPublisherUnavailable", err)
	}
	if err := (Event{}).validate(); !errors.Is(err, ErrEventIDRequired) {
		t.Fatalf("validate() error = %v, want ErrEventIDRequired", err)
	}
}
