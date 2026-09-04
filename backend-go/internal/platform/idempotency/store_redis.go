package idempotency

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
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

func redisKey(scope, key string) string { return "idem:" + scope + ":" + key }

func marshalRecord(r Record) ([]byte, error) { return json.Marshal(r) }

func unmarshalRecord(b string) (Record, bool) {
	var r Record
	if err := json.Unmarshal([]byte(b), &r); err != nil {
		return Record{}, false
	}
	return r, true
}

func (s *RedisStore) Get(scope, key string) (Record, bool) {
	b, err := s.rdb.Get(context.Background(), redisKey(scope, key)).Result()
	if err != nil {
		return Record{}, false
	}
	return unmarshalRecord(b)
}

func (s *RedisStore) Reserve(scope, key string, expiresAt time.Time) (Record, bool) {
	ctx := context.Background()
	k := redisKey(scope, key)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	processing := Record{Key: key, Scope: scope, Status: StatusProcessing, ExpiresAt: expiresAt}
	payload, err := marshalRecord(processing)
	if err != nil {
		return Record{}, false
	}
	// SET NX：key 不存在 → 写入 processing 并声明执行权；已存在 → 取出现有记录。
	ok, err := s.rdb.SetNX(ctx, k, payload, ttl).Result()
	if err != nil {
		return Record{}, false
	}
	if ok {
		return processing, true
	}
	b, err := s.rdb.Get(ctx, k).Result()
	if err != nil {
		return Record{}, false
	}
	existing, _ := unmarshalRecord(b)
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
	ttl := time.Until(record.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	_ = saveCompletedScript.Run(context.Background(), s.rdb, []string{redisKey(record.Scope, record.Key)},
		payload, ttl.Milliseconds()).Err()
}

func (s *RedisStore) Delete(scope, key string) {
	s.rdb.Del(context.Background(), redisKey(scope, key))
}
