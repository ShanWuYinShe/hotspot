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
	"hotspot/backend-go/internal/platform/worker"
)

const (
	redisChannelPrefix  = "hotspot:"
	redisChannelUser    = "user:"
	redisChannelUserSub = "usersub:"
	redisChannelGroup   = "group:"
	redisChannelSession = "session:"
	redisChannelAll     = "all"
	// 订阅撤销控制通道（跨实例同步强制摘除，见 Hub.RevokeGroupSubscription / DropGroup）。
	redisChannelRevoke    = "revoke:"
	redisChannelGroupDrop = "groupdrop:"
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
	sub      *redis.PubSub
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

	// PSubscribe：模式订阅（SUBSCRIBE 不做 glob 匹配，字面订阅 "hotspot:*" 会静默丢失全部跨实例事件）
	sub := rdb.PSubscribe(ctx, redisChannelPrefix+"*")
	b.sub = sub
	b.subCh = sub.Channel()

	b.done.Add(1)
	// 注意用 worker.Go 而非 GoRestartable：receiveLoop 自带 defer b.done.Done()
	// 与 Close 的 done.Wait() 配对，panic 重启会造成二次 Done（负计数 panic）
	// 并破坏 Wait 语义。桥内消息处理已逐条防抖（unmarshal 失败仅记日志跳过）。
	worker.Go("realtime.redis_bridge", b.receiveLoop)
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
	_ = b.sub.Close()
	b.done.Wait()
	// 不关闭共享的 *redis.Client：同一 client 还被 resumeStore 与健康探针使用，生命周期归 bootstrap 统一管理。
	return nil
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
	// Order matters: "session:" and "usersub:" must be checked before "user:"
	// because strings.HasPrefix("usersub:xxx", "user:") is true.
	switch {
	case strings.HasPrefix(channel, redisChannelSession):
		sessionID := strings.TrimPrefix(channel, redisChannelSession)
		b.hub.closeSessionLocal(sessionID, event)
	case strings.HasPrefix(channel, redisChannelUserSub):
		userID := strings.TrimPrefix(channel, redisChannelUserSub)
		b.hub.deliverUserSubscribersLocal(userID, event)
	case strings.HasPrefix(channel, redisChannelUser):
		userID := strings.TrimPrefix(channel, redisChannelUser)
		b.hub.deliverUserLocal(userID, event)
	case strings.HasPrefix(channel, redisChannelGroupDrop):
		groupID := strings.TrimPrefix(channel, redisChannelGroupDrop)
		b.hub.dropGroupLocal(groupID)
	case strings.HasPrefix(channel, redisChannelGroup):
		groupID := strings.TrimPrefix(channel, redisChannelGroup)
		b.hub.deliverGroupLocal(groupID, event)
	case strings.HasPrefix(channel, redisChannelRevoke):
		// 订阅撤销控制事件：只清本实例注册表，不向 sink 投递。载荷经 JSON
		// 往返为 map（跨实例脏消息/空载荷按宽口径防抖，不做未检查断言——
		// nil interface 断言会 panic）。
		userID := strings.TrimPrefix(channel, redisChannelRevoke)
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			observability.L().Warn("redis bridge: revoke event with unexpected payload", "user_id", userID)
			break
		}
		groupID, ok := payload["group_id"].(string)
		if !ok || groupID == "" {
			observability.L().Warn("redis bridge: revoke event missing group_id", "user_id", userID)
			break
		}
		b.hub.revokeGroupLocal(userID, groupID)
	case channel == redisChannelAll:
		b.hub.deliverAllLocal(event)
	default:
		observability.L().Warn("redis bridge: unknown channel", "channel", channel)
	}
}
