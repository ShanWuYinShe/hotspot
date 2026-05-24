package realtime

import "context"

// EventPublisher fans out realtime events to other backend instances.
// When nil (or NopPublisher), the Hub operates in single-instance mode.
type EventPublisher interface {
	// Publish sends an event to the given channel for cross-instance delivery.
	Publish(ctx context.Context, channel string, event Event) error
	// Close shuts down the publisher and releases resources.
	Close() error
}

// NopPublisher is a no-op EventPublisher used when Redis is not configured.
type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, string, Event) error { return nil }
func (NopPublisher) Close() error                                 { return nil }
