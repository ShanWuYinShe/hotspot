package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/observability"
)

const (
	redisChannelPrefix  = "hotspot:"
	redisChannelUser    = "user:"
	redisChannelUserSub = "usersub:"
	redisChannelGroup   = "group:"
	redisChannelAll     = "all"
)

type redisEnvelope struct {
	Origin string `json:"o"`
	Event  Event  `json:"e"`
}

// RedisBridge implements EventPublisher using Redis Pub/Sub.
// It also runs a subscriber goroutine that delivers remote events
// to the local Hub via the local-only delivery methods.
type RedisBridge struct {
	rdb      *redis.Client
	instance string
	hub      *Hub
	subCh    <-chan *redis.Message
	cancel   context.CancelFunc
	closed   atomic.Bool
	done     sync.WaitGroup
}

// NewRedisBridge creates a Redis-based EventPublisher and starts a
// subscriber goroutine. Call Close() to shut it down.
func NewRedisBridge(rdb *redis.Client, instanceID string, hub *Hub) (*RedisBridge, error) {
	ctx, cancel := context.WithCancel(context.Background())
	b := &RedisBridge{
		rdb:      rdb,
		instance: instanceID,
		hub:      hub,
		cancel:   cancel,
	}

	sub := rdb.Subscribe(ctx, redisChannelPrefix+"*")
	b.subCh = sub.Channel()

	b.done.Add(1)
	go b.receiveLoop()
	return b, nil
}

func (b *RedisBridge) Publish(ctx context.Context, channel string, event Event) error {
	if b.closed.Load() {
		return nil
	}
	envelope := redisEnvelope{Origin: b.instance, Event: event}
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal redis envelope: %w", err)
	}
	return b.rdb.Publish(ctx, redisChannelPrefix+channel, data).Err()
}

func (b *RedisBridge) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil
	}
	b.cancel()
	b.done.Wait()
	return b.rdb.Close()
}

func (b *RedisBridge) receiveLoop() {
	defer b.done.Done()
	for msg := range b.subCh {
		if b.closed.Load() {
			return
		}
		var envelope redisEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
			observability.L().Warn("redis bridge: invalid envelope", "err", err)
			continue
		}
		if envelope.Origin == b.instance {
			continue
		}
		channel := strings.TrimPrefix(msg.Channel, redisChannelPrefix)
		b.dispatchLocal(channel, envelope.Event)
	}
}

func (b *RedisBridge) dispatchLocal(channel string, event Event) {
	switch {
	case strings.HasPrefix(channel, redisChannelUser):
		userID := strings.TrimPrefix(channel, redisChannelUser)
		b.hub.deliverUserLocal(userID, event)
	case strings.HasPrefix(channel, redisChannelUserSub):
		userID := strings.TrimPrefix(channel, redisChannelUserSub)
		b.hub.deliverUserSubscribersLocal(userID, event)
	case strings.HasPrefix(channel, redisChannelGroup):
		groupID := strings.TrimPrefix(channel, redisChannelGroup)
		b.hub.deliverGroupLocal(groupID, event)
	case channel == redisChannelAll:
		b.hub.deliverAllLocal(event)
	default:
		observability.L().Warn("redis bridge: unknown channel", "channel", channel)
	}
}
