package media

import (
	"context"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

// queryPurgeUserUploadSessions 账户注销级联：删除本模块归属的上传会话（media_upload_sessions）。
const queryPurgeUserUploadSessions = `DELETE FROM media_upload_sessions WHERE user_id = $1`

// ListUserStorageKeys 返回该用户全部上传会话的存储键：completed 行是被引用的
// 不可写发布键（published/...），未完成行是预签名 PUT 的临时上传键
// （uploads/...）。账户注销级联在删除会话行之前调用它取出待删对象清单
// （Service.PurgeUserData），行删除后逐条删除对象（user_purge.go）。
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
// ListUserStorageKeys 的清单后、行删除完成后逐条删除（这种分层的顺序与失败
// 语义集中写在 Service 入口，见那里的注释）。
func (r *PostgresRepository) PurgeUserData(ctx context.Context, userID string) error {
	q, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return q.Exec(ctx, queryPurgeUserUploadSessions, userID)
}

// purgeObjectDeleteBatchSize 是账户注销级联对象删除的分批粒度：单用户的上传
// 会话数没有硬上限（每次 presign 一行，completed 行不会被保留期 GC 回收），
// 分批推进避免一次性把全部删除请求压向对象存储；批内逐条串行删除（失败也继续
// 下一条），同一时刻至多一个在途请求。
const purgeObjectDeleteBatchSize = 100

// PurgeUserData 账户注销级联入口：先查询该用户全部上传会话的 storage_key，
// 再删除会话行（仓储实现），最后逐条删除存储对象。顺序与失败语义：
//   - 查询 keys 失败 / 删行失败：原样上抛，编排层中止整个注销并允许整体重试
//     （此时对象未动、行仍在，重试从完整状态重来）；
//   - 对象删除放在行删除之后：删除失败仅 Warn、不阻塞注销。行已删，残留对象
//     不再被任何行引用，也无法再经预签名 URL 访问（风险面从「账号已删内容仍
//     可被签名 URL 读取」降为存储垃圾）；账号注销完成后没有重试入口，残留对象
//     依赖存储侧按前缀运维清理兜底。
//
// 取清单与删行之间存在并发窗口（并发 complete 会改写 storage_key、并发 presign
// 会新增行）：级联第一步（auth 会话撤销）已关掉该用户的正常入口，窗口只覆盖注销
// 前已在途的极少数请求，极端情况下残留个别发布键对象——与对象删除失败同属可接受
// 的存储垃圾（行已删，无重试入口，依赖存储侧按前缀运维清理兜底）。
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
	s.deletePurgeObjects(ctx, keys)
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

// deletePurgeObjects 逐条删除注销级联的对象（幂等：provider.Delete 把 404 视为
// 已删），失败仅 Warn 并继续——行已删，残留对象只是存储垃圾，不能因此让整个
// 注销失败（重试也无法再发现这些键）。
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
