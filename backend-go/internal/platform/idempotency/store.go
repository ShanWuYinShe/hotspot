package idempotency

import (
	"container/list"
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/worker"
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusExpired    Status = "expired"
)

// sweepCadence is how often the background sweeper physically removes
// expired records from the in-memory store. Get/Reserve already apply lazy
// expiry (callers see expired as StatusExpired → 22001), but without sweeping
// the map grows unboundedly in long-lived processes.
const sweepCadence = 5 * time.Minute

// maxInMemoryRecords 是无 Redis 时进程内幂等存储的容量上限。记录本身带 TTL 且
// 有后台 sweeper，但「每次请求都换新 Idempotency-Key 的重试/扫描流量」能在一次
// 清扫周期内把 map 撑到任意大；有上限才能保证长跑进程内存有界。
const maxInMemoryRecords = 10_000

// capacitySweepInterval 是「容量已满时先物理清理过期记录」的冷却窗口：容量满后
// 每条新写入都会触发容量检查，若每次都做 O(n) 过期扫描，插入路径会退化为 O(n)。
const capacitySweepInterval = time.Minute

type Record struct {
	Key       string
	Scope     string
	Status    Status
	Response  []byte
	ExpiresAt time.Time
	// BodyHash 是请求体的 SHA-256（幂等上下文一致性校验，spec api_spec_errors.md
	// 22000「幂等上下文不匹配」）。存量记录为空时跳过比对（灰度兼容）。
	BodyHash string `json:"body_hash,omitempty"`
	// Headers 是 producer 在写响应体前设置的、需要在回放时原样重发的响应头。
	// 当前只保留 Set-Cookie：web 端 register/guest 的会话 cookie 由 producer
	// 在闭包内下发，而回放命中时 producer 不执行——不重发会让「网络超时后用
	// 同一 Idempotency-Key 重试」的客户端拿到 code=0 却没有会话（账号已建但
	// 处于未登录态）。存量记录为空 → 回放不补头（灰度兼容）。
	Headers map[string][]string `json:"headers,omitempty"`
}

type Store interface {
	Get(scope, key string) (Record, bool)
	// Reserve atomically inserts a "processing" record when none exists; it
	// returns the existing record if one is already present. This lets the
	// replay path decide replay-vs-in-flight without a separate check-then-act
	// (and without unsafe type assertions on the concrete store type).
	Reserve(scope, key string, expiresAt time.Time) (Record, bool)
	Save(record Record)
	// Delete removes a record outright — used to roll back the processing
	// marker when the producer panics, so the key isn't poisoned for its TTL.
	Delete(scope, key string)
}

// InMemoryStore 是单实例部署下的幂等存储：record 按 key 存 map，有效期由调用方
// 传入的 ExpiresAt 决定（惰性过期 + 后台 Sweep 清理），并带容量上限防止长跑进程
// 无界增长。
//
// 容量策略（records 与 order 双结构，均受 maxEntries 约束）：
//   - 正常路径：写入时若已满，先按冷却窗口物理清理过期记录（过期优先）；
//   - 仍满：按「最早写入」FIFO 淘汰（order 链表头部），最老的 key 最可能已过期，
//     也最不可能再被客户端重试命中；被淘汰的 key 退化为「从未存在」（可重新占用），
//     即容量压力下幂等保证让位于有界内存，仅在「短时间内新 key 数逼近上限」时发生。
type InMemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
	// order 记录写入先后（FIFO 淘汰顺序），elems 提供 O(1) 定位，避免记录被
	// Delete/Sweep 删除后 order 里残留 key 造成第二份无界内存。
	order             *list.List
	elems             map[string]*list.Element
	maxEntries        int
	lastCapacitySweep time.Time
}

// NewInMemoryStore 返回容量上限为 maxInMemoryRecords 的内存存储。
func NewInMemoryStore() *InMemoryStore { return newInMemoryStore(maxInMemoryRecords) }

// newInMemoryStore 允许自定义容量（maxEntries <= 0 表示不限制，仅供测试对比
// 旧行为）；生产构造统一走 NewInMemoryStore。
func newInMemoryStore(maxEntries int) *InMemoryStore {
	return &InMemoryStore{
		records:    make(map[string]Record),
		order:      list.New(),
		elems:      make(map[string]*list.Element),
		maxEntries: maxEntries,
	}
}

func composite(scope, key string) string { return scope + "::" + key }

// Get applies lazy expiry: records past their ExpiresAt are reported as
// expired (StatusExpired) even before the sweeper physically removes them.
// Callers can distinguish "曾存在但已过期" vs "从未存在" by checking
// record.Status == StatusExpired, which the replay layer maps to 22001.
func (s *InMemoryStore) Get(scope, key string) (Record, bool) {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[composite(scope, key)]
	if !ok {
		return Record{}, false
	}
	if now.After(record.ExpiresAt) {
		return Record{
			Key:       record.Key,
			Scope:     scope,
			Status:    StatusExpired,
			Response:  record.Response,
			ExpiresAt: record.ExpiresAt,
		}, true
	}
	return record, true
}

func (s *InMemoryStore) Reserve(scope, key string, expiresAt time.Time) (Record, bool) {
	k := composite(scope, key)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[k]; ok {
		if now.After(existing.ExpiresAt) {
			// 已存在但已过期：返回 StatusExpired 让上层映射为 CodeIdempotencyExpired (22001)，
			// 而非直接视为可重新占用。保留原记录直到 sweeper 清理，期间同一过期 key 的
			// 重试持续返回 22001，满足"曾存在但已过期" vs "从未存在"的区分。
			return Record{
				Key:       existing.Key,
				Scope:     scope,
				Status:    StatusExpired,
				Response:  existing.Response,
				ExpiresAt: existing.ExpiresAt,
			}, false
		}
		return existing, false
	}
	rec := Record{Key: key, Scope: scope, Status: StatusProcessing, ExpiresAt: expiresAt}
	s.addLocked(k, rec, now)
	return rec, true
}

func (s *InMemoryStore) Save(record Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := composite(record.Scope, record.Key)
	if _, ok := s.records[k]; !ok {
		// 新 key 才受容量约束；覆写已有记录保持原有淘汰位置（completed 覆写
		// processing 不应让记录「变成新写入」，否则 hot key 会因反复 Save 被误淘汰）。
		s.addLocked(k, record, time.Now())
		return
	}
	s.records[k] = record
}

// Delete removes the record immediately (no lazy expiry), so a retried
// Reserve can claim the key again.
func (s *InMemoryStore) Delete(scope, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(composite(scope, key))
}

// Sweep physically removes records whose ExpiresAt is in the past. Intended
// to be called by the background sweeper started via StartSweeper.
func (s *InMemoryStore) Sweep(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepExpiredLocked(now)
	s.lastCapacitySweep = now
}

// addLocked 插入一条新记录并维护 FIFO 顺序；调用方必须持有写锁。
func (s *InMemoryStore) addLocked(key string, record Record, now time.Time) {
	s.ensureCapacityLocked(now)
	s.records[key] = record
	s.elems[key] = s.order.PushBack(key)
}

// removeLocked 同步删除 map 与 FIFO 链表（保持 order/elems 与 records 一致，
// 否则内存上限只约束了 map，残留的链表节点会继续增长）。
func (s *InMemoryStore) removeLocked(key string) {
	delete(s.records, key)
	if el, ok := s.elems[key]; ok {
		s.order.Remove(el)
		delete(s.elems, key)
	}
}

// sweepExpiredLocked 物理清理过期记录；调用方必须持有写锁。
func (s *InMemoryStore) sweepExpiredLocked(now time.Time) {
	for k, rec := range s.records {
		if now.After(rec.ExpiresAt) {
			s.removeLocked(k)
		}
	}
}

// ensureCapacityLocked 在插入新 key 前腾出空间：先按冷却窗口清理过期记录
// （过期优先，尊重不同端点的 TTL 差异），仍满则按最早写入淘汰。调用方必须持有
// 写锁。
func (s *InMemoryStore) ensureCapacityLocked(now time.Time) {
	if s.maxEntries <= 0 || len(s.records) < s.maxEntries {
		return
	}
	if now.Sub(s.lastCapacitySweep) >= capacitySweepInterval {
		s.sweepExpiredLocked(now)
		s.lastCapacitySweep = now
	}
	for len(s.records) >= s.maxEntries {
		front := s.order.Front()
		if front == nil {
			// 理论不可达（records 与 order 同步维护）；兜底避免死循环。
			return
		}
		s.removeLocked(front.Value.(string))
	}
}

// StartSweeper 为指定 Store 启动后台清扫（InMemoryStore 支持 Sweep；Redis
// 由服务端 TTL 清理，直接返回 no-op stop）。显式传入 Store，不再依赖全局
// store 单例；返回的 stop 幂等，可安全在测试/停机时调用。
func StartSweeper(ctx context.Context, store Store) (stop func()) {
	sweeper, ok := store.(interface{ Sweep(time.Time) })
	if !ok {
		return func() {}
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	worker.GoRestartable("idempotency.sweeper", func() { runSweeper(sweepCtx, sweeper) })
	return cancel
}

func runSweeper(ctx context.Context, store interface{ Sweep(time.Time) }) {
	ticker := time.NewTicker(sweepCadence)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			store.Sweep(time.Now())
		}
	}
}
