package idempotency

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/observability"
)

// RedisStore 把幂等记录放到共享 Redis：多实例部署下同一 Idempotency-Key
// 无论落在哪个实例都只执行一次（进程内 InMemoryStore 各自计数，多实例失效）。
// Reserve 的原子性由 SET NX（仅当 key 不存在时写入 processing 记录）保证；
// Save 的「processing → completed」覆写由 Lua 脚本在服务端原子完成。
// Record 以 JSON 序列化存储，TTL = 记录的 ExpiresAt，到期 Redis 自动删除，
// 与内存版的惰性过期语义一致。
type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore { return &RedisStore{rdb: rdb} }

// expiredRecordGrace 是记录过期后仍保留的「墓碑」窗口：Redis 的 TTL 一旦到期
// 键即消失，Reserve 会把同一个旧 key 当作全新请求再次执行；而单实例内存实现与
// spec api_spec_errors.md 约定「曾存在但已过期」必须返回 22001（换新 key 重试），
// 不能静默重新执行。保留窗口让 Redis 实现也能识别过期命中（多实例语义一致）。
// 墓碑由 Redis TTL 自动清理，内存占用有界。
const expiredRecordGrace = 10 * time.Minute

// recordTTL 返回记录应写入 Redis 的 TTL：业务有效期 + 过期墓碑窗口。
func recordTTL(expiresAt time.Time) time.Duration {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	return ttl + expiredRecordGrace
}

// expiredRecord 把已过期记录还原为 StatusExpired（与 InMemoryStore.Get/Reserve
// 的惰性过期口径一致，replay 层据此返回 22001）。
func expiredRecord(record Record) Record {
	return Record{
		Key:       record.Key,
		Scope:     record.Scope,
		Status:    StatusExpired,
		Response:  record.Response,
		ExpiresAt: record.ExpiresAt,
	}
}

func redisKey(scope, key string) string { return "idem:" + scope + ":" + key }

func marshalRecord(r Record) ([]byte, error) { return json.Marshal(r) }

func unmarshalRecord(b string) (Record, bool) {
	var r Record
	if err := json.Unmarshal([]byte(b), &r); err != nil {
		return Record{}, false
	}
	return r, true
}

// redisCallCtx 是幂等存储每次 Redis 调用的显式超时（对齐 ratelimit/resume
// 的 2s 模式）：此前用裸 context.Background()，仅靠 client 级超时兜底，
// 与其它共享 Redis 的模块超时语义不一致。
func redisCallCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func (s *RedisStore) Get(scope, key string) (Record, bool) {
	ctx, cancel := redisCallCtx()
	defer cancel()
	b, err := s.rdb.Get(ctx, redisKey(scope, key)).Result()
	if err != nil {
		return Record{}, false
	}
	record, ok := unmarshalRecord(b)
	if !ok {
		return Record{}, false
	}
	if time.Now().After(record.ExpiresAt) {
		return expiredRecord(record), true
	}
	return record, true
}

func (s *RedisStore) Reserve(scope, key string, expiresAt time.Time) (Record, bool) {
	ctx, cancel := redisCallCtx()
	defer cancel()
	k := redisKey(scope, key)
	// TTL 覆盖「业务有效期 + 墓碑窗口」：过期后记录仍在，供 22001 识别。
	ttl := recordTTL(expiresAt)
	processing := Record{Key: key, Scope: scope, Status: StatusProcessing, ExpiresAt: expiresAt}
	payload, err := marshalRecord(processing)
	if err != nil {
		return Record{}, false
	}
	// SET NX：key 不存在 → 写入 processing 并声明执行权；已存在 → 取出现有记录。
	ok, err := s.rdb.SetNX(ctx, k, payload, ttl).Result()
	if err != nil {
		// fail-open（与 login tracker 同口径）：Redis 抖动时放弃幂等保证
		// 直通执行，优于让全部带幂等键的写端点整体不可用（22002 拒服）。
		// 代价是该 key 在故障窗口内失去去重（重复执行由业务侧幂等性兜底）。
		observability.L().Warn("idempotency redis reserve failed, failing open", "error", err)
		return processing, true
	}
	if ok {
		return processing, true
	}
	b, err := s.rdb.Get(ctx, k).Result()
	if err != nil {
		// 同上：已存在 key 但读不出（故障窗口），直通执行。
		observability.L().Warn("idempotency redis get failed, failing open", "error", err)
		return processing, true
	}
	existing, ok := unmarshalRecord(b)
	if !ok {
		return Record{}, false
	}
	// 与内存实现一致：曾存在但已过期 → StatusExpired（22001），而不是允许复用旧 key。
	if time.Now().After(existing.ExpiresAt) {
		return expiredRecord(existing), false
	}
	return existing, false
}

// saveCompletedScript 原子覆写：仅 processing 记录可被完成态响应覆盖（与
// InMemoryStore.Save 的语义一致——重复 Save 幂等）。
var saveCompletedScript = redis.NewScript(`
local cur = redis.call('GET', KEYS[1])
if cur then
	local rec = cjson.decode(cur)
	if rec.Status ~= 'completed' then
		redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
	end
	return
end
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
`)

func (s *RedisStore) Save(record Record) {
	payload, err := marshalRecord(record)
	if err != nil {
		return
	}
	// TTL 同样覆盖墓碑窗口：完成态记录过期后仍需可识别为 22001。
	ttl := recordTTL(record.ExpiresAt)
	ctx, cancel := redisCallCtx()
	defer cancel()
	_ = saveCompletedScript.Run(ctx, s.rdb, []string{redisKey(record.Scope, record.Key)},
		payload, ttl.Milliseconds()).Err()
}

func (s *RedisStore) Delete(scope, key string) {
	ctx, cancel := redisCallCtx()
	defer cancel()
	s.rdb.Del(ctx, redisKey(scope, key))
}
