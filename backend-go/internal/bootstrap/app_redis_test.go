package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
)

// app_redis_test.go — Redis 启动探测三态（未配置/不可用/可用）、启动降级可见性
// （错误信息脱敏 + 指标）与 App 停机关闭共享 Redis client 的回归测试。

// TestNewRedisClientNotConfigured：未配置 REDIS_URL 是显式支持的单实例模式，
// 返回 (nil, nil)，由调用方打 Info 级说明（不能报错/告警）。
func TestNewRedisClientNotConfigured(t *testing.T) {
	client, err := newRedisClient(config.Config{})
	if client != nil || err != nil {
		t.Fatalf("expected (nil, nil) without REDIS_URL, got client=%v err=%v", client, err)
	}
}

// TestNewRedisClientAvailable：可达时返回可用 client（miniredis 真实 Redis 协议）。
func TestNewRedisClientAvailable(t *testing.T) {
	mr := miniredis.RunT(t)
	client, err := newRedisClient(config.Config{RedisURL: "redis://" + mr.Addr() + "/0"})
	if err != nil || client == nil {
		t.Fatalf("expected live redis client, got client=%v err=%v", client, err)
	}
	t.Cleanup(func() { _ = client.Close() })
}

// TestNewRedisClientUnreachable：显式配置但不可达必须返回非空错误（调用方据此
// 打 Error 级降级日志），且 client 被丢弃（不让半死连接拖慢热路径）。
func TestNewRedisClientUnreachable(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close() // 关闭后连接被拒绝，模拟 Redis 宕机/网络不可达
	client, err := newRedisClient(config.Config{RedisURL: "redis://" + addr + "/0"})
	if err == nil {
		t.Fatal("expected error for unreachable redis")
	}
	if client != nil {
		t.Fatal("expected client to be discarded on ping failure")
	}
}

// TestNewRedisClientInvalidURLDoesNotLeakPassword：URL 解析失败时 net/url 的
// *url.Error 会回显完整 URL（含口令），返回的错误必须已脱敏（口令不得出现在
// 日志/告警里）。
func TestNewRedisClientInvalidURLDoesNotLeakPassword(t *testing.T) {
	const password = "super-secret-redis-pass"
	client, err := newRedisClient(config.Config{RedisURL: "redis://:" + password + "@127.0.0.1:not-a-port/0"})
	if err == nil {
		t.Fatal("expected parse error for invalid redis url")
	}
	if client != nil {
		t.Fatal("expected nil client for invalid redis url")
	}
	if strings.Contains(err.Error(), password) {
		t.Fatalf("error leaked redis password: %v", err)
	}
}

// TestRedisEndpointRedactsCredentials：日志用的端点描述只保留 scheme/host/db。
func TestRedisEndpointRedactsCredentials(t *testing.T) {
	endpoint := redisEndpoint("redis://:super-secret@redis.internal:6379/2")
	if endpoint != "redis://redis.internal:6379/2" {
		t.Fatalf("unexpected redacted endpoint: %q", endpoint)
	}
	if strings.Contains(endpoint, "super-secret") {
		t.Fatalf("endpoint leaked password: %q", endpoint)
	}
	if got := redisEndpoint("://bad-url"); strings.Contains(got, "bad-url") {
		t.Fatalf("invalid url must not be echoed back, got %q", got)
	}
}

// TestRedisStartupDegradedGauge：降级状态用 Gauge 暴露（0/1），便于
// hotspot_redis_startup_degraded == 1 的告警稳定命中。
func TestRedisStartupDegradedGauge(t *testing.T) {
	render := func(degraded bool) string {
		registry := prometheus.NewRegistry()
		registry.MustRegister(newRedisStartupDegradedGauge(degraded))
		rr := httptest.NewRecorder()
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).
			ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		return rr.Body.String()
	}
	if body := render(true); !strings.Contains(body, "hotspot_redis_startup_degraded 1") {
		t.Fatalf("expected degraded gauge 1, got:\n%s", body)
	}
	if body := render(false); !strings.Contains(body, "hotspot_redis_startup_degraded 0") {
		t.Fatalf("expected available gauge 0, got:\n%s", body)
	}
}

// TestRedisReadinessProbeStates：就绪探针的 Redis 三态——未配置跳过（ok）、
// 可用实时 Ping、配置但启动探测失败持续 degraded（required=false 仍 200，
// 限流/幂等 fail-open 不摘流量）。启动失败后 client 已丢弃且不会重连，探针
// 不能误报全绿。
func TestRedisReadinessProbeStates(t *testing.T) {
	if probe := redisReadinessProbe(nil, nil); probe != nil {
		t.Fatal("nil redis client without startup error must yield nil ping (treated as ok)")
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	live := redisReadinessProbe(rdb, nil)
	if live == nil {
		t.Fatal("expected live probe for available redis client")
	}
	if err := live(context.Background()); err != nil {
		t.Fatalf("expected live redis probe to pass, got %v", err)
	}

	startupErr := errors.New("redis startup ping failed")
	failed := redisReadinessProbe(nil, startupErr)
	if failed == nil {
		t.Fatal("configured-but-unavailable redis must still be probed")
	}
	if err := failed(context.Background()); !errors.Is(err, startupErr) {
		t.Fatalf("expected startup error to persist on probe, got %v", err)
	}
	code, body := probeResponse(t, newReadinessProbe(
		readinessCheck{name: "database", required: true, ping: okPing},
		readinessCheck{name: "redis", required: false, ping: failed},
	))
	if code != 200 || body["status"] != "degraded" {
		t.Fatalf("expected degraded 200 for startup redis failure, got code=%d body=%#v", code, body)
	}
}

// shutdownOrderDB 记录停机时序的 DB 桩：Close 时探测共享 Redis client 是否
// 已经关闭，用于锁定「Redis 先于 DB 关闭」的顺序契约。
type shutdownOrderDB struct {
	redis *redis.Client

	mu                   sync.Mutex
	closed               bool
	redisClosedAtDBClose bool
	redisPingErr         string
}

func (db *shutdownOrderDB) Ping(context.Context) error { return nil }

func (db *shutdownOrderDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.closed = true
	if db.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := db.redis.Ping(ctx).Err(); err != nil {
			db.redisClosedAtDBClose = true
			db.redisPingErr = err.Error()
		}
	}
	return nil
}

// TestAppShutdownClosesRedisBeforeDatabase：优雅停机必须显式关闭共享 Redis
// client（bridge.Close 只退订阅、不关共享 client），且顺序在 DB 池回收之前。
func TestAppShutdownClosesRedisBeforeDatabase(t *testing.T) {
	logger, err := observability.NewLogger("test", "error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	db := &shutdownOrderDB{redis: rdb}
	app := &App{
		logger:      logger,
		hub:         realtime.NewHub(),
		redis:       rdb,
		persistence: &persistence.Factory{DB: db},
	}

	app.Shutdown()

	db.mu.Lock()
	closed := db.closed
	redisClosedAtDBClose := db.redisClosedAtDBClose
	pingErr := db.redisPingErr
	db.mu.Unlock()
	if !closed {
		t.Fatal("expected database to be closed during shutdown")
	}
	if !redisClosedAtDBClose {
		t.Fatal("expected redis client to be closed before database pool")
	}
	if !strings.Contains(pingErr, "client is closed") {
		t.Fatalf("expected closed redis client error at db close, got %q", pingErr)
	}
	// 停机后 client 必须处于关闭态（Ping 报 client is closed，而非继续可用）。
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err == nil || !strings.Contains(err.Error(), "client is closed") {
		t.Fatalf("expected redis client to be closed after shutdown, got %v", err)
	}
}

// TestAppShutdownWithoutRedis：未配置 Redis 的 App（开发/单机模式）停机时
// 不得因 nil client 报错或 panic（保持原有 bootstrap 关闭路径不回归）。
func TestAppShutdownWithoutRedis(t *testing.T) {
	logger, err := observability.NewLogger("test", "error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	app := &App{
		logger:      logger,
		hub:         realtime.NewHub(),
		persistence: &persistence.Factory{DB: persistence.NewInMemoryDB()},
	}
	app.Shutdown()
}
