package media

import (
	"context"
	"fmt"
	"sync"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/outbox"
	"hotspot/backend-go/internal/platform/persistence"
)

// queryPurgeUserUploadSessions 账户注销级联：删除本模块归属的上传会话（media_upload_sessions）。
const queryPurgeUserUploadSessions = `DELETE FROM media_upload_sessions WHERE user_id = $1`

// ListUserStorageKeys 返回该用户全部上传会话的存储键：completed 行是被引用的
// 不可写发布键（published/...），未完成行是预签名 PUT 的临时上传键
// （uploads/...）。账户注销级联在删除会话行之前调用它取出待删对象清单
// （Service.PurgeUserData），行删除后按批投递 outbox 异步删除对象（user_purge.go）。
func (r *PostgresRepository) ListUserStorageKeys(ctx context.Context, userID string) ([]string, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := queryable.Query(ctx, queryListUserStorageKeys, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var storageKey string
		if err := rows.Scan(&storageKey); err != nil {
			return nil, err
		}
		keys = append(keys, storageKey)
	}
	return keys, rows.Err()
}

// PurgeUserData 账户注销级联（users 模块经 bootstrap 编排调用）：删除本模块
// 归属的该用户数据；幂等，失败由编排层中止并允许整体重试。
// 本方法只负责会话行删除——存储对象由 Service.PurgeUserData 在取得
// ListUserStorageKeys 的清单后、行删除完成后按批投递 outbox 异步删除（这种
// 分层的顺序与失败语义集中写在 Service 入口，见那里的注释）。
func (r *PostgresRepository) PurgeUserData(ctx context.Context, userID string) error {
	q, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return q.Exec(ctx, queryPurgeUserUploadSessions, userID)
}

// purgeObjectDeleteBatchSize 是账户注销级联「同步回退」对象删除的分批粒度：
// 单用户的上传会话数没有硬上限（每次 presign 一行，completed 行不会被保留期
// GC 回收），分批推进避免一次性把全部删除请求压向对象存储；批内逐条串行删除
// （失败也继续下一条），同一时刻至多一个在途请求。异步路径的删除由 outbox
// dispatcher 投递（每个事件一批），此常量只约束未装配/入队失败的降级路径。
const purgeObjectDeleteBatchSize = 100

// purgeObjectEventBatchSize 是注销级联对象清理事件的单事件键数上限：单事件
// ≤200 个 key 既限制单次执行时长（handler 在 outbox 事务外执行，单事件超时
// 60s），也让重试粒度足够细（失败只重删这一批，已删对象 404 幂等）。
const purgeObjectEventBatchSize = 200

// purgeDeleteChunkSize/purgeDeleteConcurrency 限制单事件内对象删除的并发与
// 分批：handler 对每批 key 最多 purgeDeleteConcurrency 条 DELETE 并发，批间
// 检查 ctx（dispatcher 的单事件超时/停机），避免 200 键全串行时单事件最坏
// 时长 200×单条 10s 超时=2000s，也避免一次性向对象存储打出突发请求。
const (
	purgeDeleteChunkSize   = 50
	purgeDeleteConcurrency = 4
)

// PurgeObjectsPayload 是 outbox.TopicMediaPurge 的事件 payload（media 生产、
// bootstrap dispatcher 消费）：user_id 仅用于日志与排查，keys 是该用户一批
// 待删对象键（对应的上传会话行已在入队前删除）。
type PurgeObjectsPayload struct {
	UserID string   `json:"user_id"`
	Keys   []string `json:"keys"`
}

// outboxWriter 是平台 outbox.Writer 的最小消费接口（bootstrap 注入）。
type outboxWriter interface {
	Enqueue(ctx context.Context, topic string, payload any) error
}

// SetEventOutbox 注入账户注销对象清理的 outbox writer（bootstrap 装配；未注入
// 时退化为同步删除，见 dispatchPurgeObjects）。
func (s *Service) SetEventOutbox(w outboxWriter) { s.eventOutbox = w }

// PurgeUserData 账户注销级联入口：先查询该用户全部上传会话的 storage_key，
// 再删除会话行（仓储实现），最后把存储对象清理按批投递 outbox 异步执行。
// 顺序与失败语义：
//   - 查询 keys 失败 / 删行失败：原样上抛，编排层中止整个注销并允许整体重试
//     （此时对象未动、行仍在，重试从完整状态重来）；
//   - 对象清理放在行删除之后：入队本身不阻塞注销（未入队的键回退同步删除）。
//     行已删，残留对象不再被任何行引用，也无法再经预签名 URL 访问（风险面从
//     「账号已删内容仍可被签名 URL 读取」降为存储垃圾）；入队后若投递失败由
//     outbox 的重试/死信兜底，比注销请求内重试更有恢复能力。
//
// 「行删除 → 入队」之间进程崩溃会让对象失去清理入口（与同步实现在删行后崩溃
// 同属可接受的存储垃圾）：不能在删行前入队——那样删行失败时 dispatcher 会把
// 仍被行引用的对象删掉，破坏「行删除后对象不再被引用」的安全语义。
//
// 取清单与删行之间存在并发窗口（并发 complete 会改写 storage_key、并发 presign
// 会新增行）：级联第一步（auth 会话撤销）已关掉该用户的正常入口，窗口只覆盖注销
// 前已在途的极少数请求，极端情况下残留个别发布键对象——与对象删除失败同属可接受
// 的存储垃圾（行已删，依赖存储侧按前缀运维清理兜底）。
//
// 内存/测试仓储无跨模块行可清时（不实现 userDataPurger）为 no-op；不实现
// userStorageKeyLister 的仓储退化为仅删行（没有可删的对象键）。
func (s *Service) PurgeUserData(ctx context.Context, userID string) error {
	purger, ok := s.repo.(userDataPurger)
	if !ok {
		return nil
	}
	keys, err := s.listPurgeStorageKeys(ctx, userID)
	if err != nil {
		return err
	}
	if err := purger.PurgeUserData(ctx, userID); err != nil {
		return err
	}
	s.dispatchPurgeObjects(ctx, userID, keys)
	return nil
}

// listPurgeStorageKeys 在删行前取回待删对象清单（Postgres 仓储实现
// userStorageKeyLister；内存/测试仓储无此能力时返回空清单）。
func (s *Service) listPurgeStorageKeys(ctx context.Context, userID string) ([]string, error) {
	lister, ok := s.repo.(userStorageKeyLister)
	if !ok {
		return nil, nil
	}
	keys, err := lister.ListUserStorageKeys(ctx, userID)
	if err != nil {
		observability.FromContext(ctx).Error("account purge media list storage keys failed", "error", err, "user_id", userID)
		return nil, err
	}
	return keys, nil
}

// dispatchPurgeObjects 在会话行删除后清理对象：优先按批入队 outbox（每个事件
// ≤purgeObjectEventBatchSize 个键），由 dispatcher 在请求生命周期外投递；未注入
// writer（单测/内存装配）或入队失败时回退同步删除，保证不因异步化产生孤儿。
// 入队失败只回退「尚未入队」的剩余键：已入队批次由 dispatcher 负责，重复删除
// 虽然幂等但没必要。
func (s *Service) dispatchPurgeObjects(ctx context.Context, userID string, storageKeys []string) {
	keys := compactStorageKeys(storageKeys)
	if len(keys) == 0 {
		return
	}
	logger := observability.FromContext(ctx)
	if s.eventOutbox == nil {
		// 单测/内存装配没有 outbox：保持注销级联既有的同步删除语义。
		logger.Warn("account purge media outbox not configured, deleting objects synchronously",
			"user_id", userID, "objects", len(keys))
		s.deletePurgeObjects(ctx, keys)
		return
	}
	enqueuedObjects, events := 0, 0
	for start := 0; start < len(keys); start += purgeObjectEventBatchSize {
		end := start + purgeObjectEventBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		event := PurgeObjectsPayload{UserID: userID, Keys: keys[start:end]}
		if err := s.eventOutbox.Enqueue(ctx, outbox.TopicMediaPurge, event); err != nil {
			// 入队失败（DB 不可用/无持久化）时剩余键同步删除：异步化不能把
			// 「行已删」变成「对象永久孤儿」。
			logger.Warn("account purge media outbox enqueue failed, falling back to synchronous delete",
				"error", err, "user_id", userID, "objects", len(keys),
				"enqueued_objects", enqueuedObjects, "remaining_objects", len(keys)-start)
			s.deletePurgeObjects(ctx, keys[start:])
			return
		}
		enqueuedObjects = end
		events++
	}
	logger.Info("account purge media objects enqueued", "user_id", userID, "objects", len(keys), "events", events)
}

// compactStorageKeys 去掉空 storage_key（防御：不向对象存储发空键请求，也不把
// 空键写进 outbox payload）。
func compactStorageKeys(storageKeys []string) []string {
	keys := make([]string, 0, len(storageKeys))
	for _, storageKey := range storageKeys {
		if storageKey == "" {
			continue
		}
		keys = append(keys, storageKey)
	}
	return keys
}

// DeletePurgeObjects 是 outbox.TopicMediaPurge 的处理逻辑（bootstrap 以
// RegisterExternal 注册，dispatcher 在领取事务之外、带单事件超时执行）：
// 按 payload 的键分批删除对象（provider.Delete 把 404 视为已删），空 keys 直接
// 成功。单条失败继续删其余对象（重试只需处理失败键），最后返回错误让 outbox
// 按既有指数退避/死信语义重试。批内并发上限 purgeDeleteConcurrency、批间响应
// ctx 取消（单事件总超时/停机），避免单事件最坏时长被串行 10s 超时放大到
// 2000s。日志带 user_id 与数量。
func (s *Service) DeletePurgeObjects(ctx context.Context, userID string, storageKeys []string) error {
	keys := compactStorageKeys(storageKeys)
	if len(keys) == 0 {
		return nil
	}
	logger := observability.FromContext(ctx)
	var deletedObjects, failedObjects int
	var firstErr error
	for start := 0; start < len(keys); start += purgeDeleteChunkSize {
		if err := ctx.Err(); err != nil {
			// 单事件总超时/停机：不再发起新批，剩余键留在 payload 里由 outbox
			// 重试（已删对象 404 幂等）。
			failedObjects += len(keys) - start
			firstErr = err
			break
		}
		end := min(start+purgeDeleteChunkSize, len(keys))
		deleted, failed, batchErr := s.deletePurgeObjectsChunk(ctx, userID, keys[start:end], logger)
		deletedObjects += deleted
		failedObjects += failed
		if firstErr == nil && batchErr != nil {
			firstErr = batchErr
		}
	}
	if firstErr != nil {
		logger.Warn("account purge media object batch incomplete",
			"error", firstErr, "user_id", userID, "objects", len(keys), "failed_objects", failedObjects)
		return fmt.Errorf("media: purge objects for user %s: %d/%d failed: %w", userID, failedObjects, len(keys), firstErr)
	}
	logger.Info("account purge media objects deleted",
		"user_id", userID, "objects", len(keys), "deleted_objects", deletedObjects)
	return nil
}

// deletePurgeObjectsChunk 删除一批对象：批内最多 purgeDeleteConcurrency 条
// DELETE 并发（单条失败继续删其余，已删 404 幂等），返回本批成功/失败数与
// 首个错误。
func (s *Service) deletePurgeObjectsChunk(ctx context.Context, userID string, storageKeys []string, logger *observability.Logger) (deleted, failed int, firstErr error) {
	sem := make(chan struct{}, purgeDeleteConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, storageKey := range storageKeys {
		sem <- struct{}{}
		wg.Add(1)
		go func(storageKey string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.provider.Delete(ctx, storageKey); err != nil {
				mu.Lock()
				failed++
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				logger.Warn("account purge media object delete failed",
					"error", err, "user_id", userID, "storage_key", storageKey)
				return
			}
			mu.Lock()
			deleted++
			mu.Unlock()
		}(storageKey)
	}
	wg.Wait()
	return deleted, failed, firstErr
}

// deletePurgeObjects 同步回退：逐条删除注销级联的对象（幂等：provider.Delete 把
// 404 视为已删），失败仅 Warn 并继续——行已删，残留对象只是存储垃圾，不能因此
// 让整个注销失败（重试也无法再发现这些键）。仅用于未装配 outbox（单测/内存
// 装配）或 Enqueue 失败的回退路径。
func (s *Service) deletePurgeObjects(ctx context.Context, storageKeys []string) {
	if len(storageKeys) == 0 {
		return
	}
	logger := observability.FromContext(ctx)
	var deletedObjects, failedObjects int
	for start := 0; start < len(storageKeys); start += purgeObjectDeleteBatchSize {
		end := start + purgeObjectDeleteBatchSize
		if end > len(storageKeys) {
			end = len(storageKeys)
		}
		for _, storageKey := range storageKeys[start:end] {
			if storageKey == "" {
				continue
			}
			if err := s.provider.Delete(ctx, storageKey); err != nil {
				failedObjects++
				logger.Warn("account purge media object delete failed", "error", err, "storage_key", storageKey)
				continue
			}
			deletedObjects++
		}
	}
	logger.Info("account purge media objects",
		"objects", len(storageKeys), "deleted_objects", deletedObjects, "failed_objects", failedObjects)
}

type userDataPurger interface {
	PurgeUserData(ctx context.Context, userID string) error
}

// userStorageKeyLister 是 Postgres 仓储额外提供的能力：注销级联删行前取回待删
// 对象清单（内存/测试仓储不实现，Service 退化为仅删行）。
type userStorageKeyLister interface {
	ListUserStorageKeys(ctx context.Context, userID string) ([]string, error)
}
