package media

import (
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/worker"
)

// 数据保留策略（成熟度审计：media_upload_sessions 未完成会话与对应孤儿
// 对象此前永不清理——应用代码零对象删除调用，媒体只进不出）。
// 模式对齐 notify StartPruneSweeper。
const (
	retentionSweepCadence = time.Hour
	// 未完成上传会话保留 2 天：预签名 URL 短时有效，正常上传在分钟级完成，
	// 2 天覆盖慢网/中断重试窗口，同时把「弃置会话占用存储」的窗口从 7 天压到
	// 2 天（配合 Service.checkPresignQuota 的 presign 配额，共同收敛单账号的
	// 存储放大面；7 天 × 无限弃置是第 81 轮审计的 DoS 残留）。
	staleUploadRetentionDays = 2
	// 单轮最多处理条数：500 条/轮 × 每小时一轮，足以消化配额时代遗留的弃置
	// 会话积压；同时限制单轮时长（每条约 1 次 S3 delete + 1 次 DB delete），
	// 避免单轮长时间占用 sweeping goroutine。
	staleUploadBatchLimit = 500
)

// uploads/ 前缀孤儿对象清理（第 82 轮审计 P2-2）：complete 成功后服务端尽力
// 删除旧上传键，但预签名 PUT URL 在 TTL（mediaPresignTTL，15 分钟）内仍然有效，
// 客户端凭原 URL 重 PUT 即可把 uploads/ 键写回来——完成态行的 storage_key 已改
// 写为 published/ 键、保留期 GC 只扫未完成行、注销级联只删当前 storage_key，
// 该对象因此永远无人回收。这里按「对象早于保留期（2 天）且 DB 无对应
// storage_key 行」判定孤儿并删除：2 天与未完成会话保留期同口径，覆盖「已直传、
// complete 尚未提交」的正常在途窗口；引用判定覆盖任意状态的行（含 copy-on-
// complete 之前的历史完成态行，其 storage_key 仍是 uploads/ 键，有行时必须保留）。
//
// 单轮有界：最多翻 orphanUploadListMaxPages 页（每页 orphanUploadListPageSize
// 条）、最多删 orphanUploadDeleteLimit 条；删除失败只告警，下一轮重试（对象仍
// 满足「无行 + 旧」条件）。已知限制：列举按对象键字典序分页、每轮从首页开始，
// 若前缀对象数超过扫描窗口且窗口内全是仍有引用的键，窗口外对象要等窗口内对象
// 被回收后才会进入扫描；uploads/ 前缀的稳态规模（未完成会话 ≤20/账号 + 2 天内
// 孤儿）远小于窗口上界，当前无需持久化游标（会引入迁移/状态面）。
const (
	// orphanUploadObjectPrefix 是客户端直传对象键前缀（storageKeyFor 生成
	// uploads/<userID>/<uploadID>.<ext>），只清这个前缀，不碰 published/ 业务数据。
	orphanUploadObjectPrefix = "uploads/"
	// orphanUploadListPageSize 是单次 ListObjectsV2 请求的 max-keys：500 条/页
	// 让单页响应体与 DB 批量引用查询（ANY 500 键）都保持轻量。
	orphanUploadListPageSize = 500
	// orphanUploadListMaxPages 是单轮翻页上限：20 页 = 每轮最多扫描 1 万个对象。
	orphanUploadListMaxPages = 20
	// orphanUploadDeleteLimit 是单轮删除上限（与 staleUploadBatchLimit 同量级）：
	// 保证单轮时长有界，剩余对象下一轮继续。
	orphanUploadDeleteLimit = 500
	// orphanUploadSweepTimeout 是孤儿清理阶段的整体预算（含列举、引用查询、删除）：
	// 超出后剩余对象留待下一轮，避免拖住 sweeper 循环。
	orphanUploadSweepTimeout = 60 * time.Second
)

var (
	retentionSweepMu      sync.Mutex
	retentionSweepStarted bool
)

// StartRetentionSweeper 启动保留期清理，每轮两个阶段：
//  1. 超期未完成会话：先删其可能已部分上传的存储对象（幂等，404 视为已删），
//     再删会话行；
//  2. uploads/ 前缀孤儿对象：无会话行引用且超过保留期的对象（见
//     sweepOrphanUploadObjects）。
//
// goroutine 绑定 ctx 随优雅停机退出；重复调用 no-op。
//
// ctx/ticker 必须在每次 worker 运行内部创建（见 runRetentionSweeper）：此前在
// 外层用 context.WithCancel 派生 sweepCtx、并在 worker 函数里 defer cancel，
// worker.GoRestartable 在 panic 后重新执行同一闭包时会复用已被 cancel 的
// sweepCtx——重启后的每一轮 sweep 都以 context canceled 静默失败，未完成上传
// 永不回收（第 82 轮审计 P2-4）。
func (s *Service) StartRetentionSweeper(ctx context.Context) {
	retentionSweepMu.Lock()
	defer retentionSweepMu.Unlock()
	if retentionSweepStarted {
		return
	}
	retentionSweepStarted = true
	worker.GoRestartable("media.retention_sweeper", func() {
		// 只有干净退出（ctx 取消）才复位幂等标志：panic 时 GoRestartable 会立即
		// 重启同一闭包，标志必须保持 true，否则并发调用 StartRetentionSweeper
		// 会再起一个 sweeper（重复清扫、goroutine 泄漏）。
		cleanExit := false
		defer func() {
			if !cleanExit {
				return
			}
			retentionSweepMu.Lock()
			retentionSweepStarted = false
			retentionSweepMu.Unlock()
		}()
		s.runRetentionSweeper(ctx)
		cleanExit = true
	})
}

// runRetentionSweeper 是保留期 sweeper 的单次运行周期（GoRestartable panic
// 重启时重新进入本函数）：创建 ticker、启动即先跑一轮，之后按
// retentionSweepCadence 循环直到 ctx 取消。
func (s *Service) runRetentionSweeper(ctx context.Context) {
	ticker := time.NewTicker(retentionSweepCadence)
	defer ticker.Stop()
	s.runRetentionSweep(ctx) // 启动即先跑一轮
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runRetentionSweep(ctx)
		}
	}
}

// runRetentionSweep 执行一轮保留期清理：先回收超期未完成会话（对象+行），再
// 清理 uploads/ 前缀下已无会话引用的孤儿对象（第 82 轮审计 P2-2）。两阶段互相
// 独立：会话阶段失败只影响自身，「行已删对象未删」的残留由孤儿阶段按保留期
// 兜底；孤儿阶段失败也只告警，下一轮重试。
func (s *Service) runRetentionSweep(ctx context.Context) {
	s.sweepStaleUploads(ctx)
	s.sweepOrphanUploadObjects(ctx)
}

func (s *Service) sweepStaleUploads(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cutoff := time.Now().UTC().AddDate(0, 0, -staleUploadRetentionDays)
	stale, err := s.repo.ListStaleUploads(sweepCtx, cutoff, staleUploadBatchLimit)
	if err != nil {
		observability.FromContext(ctx).Warn("media retention sweep list failed", "error", err)
		return
	}
	var deletedObjects, deletedRows int
	for _, item := range stale {
		// 对象可能从未真正上传（presign 后放弃）——404 视为已删，不算失败。
		if err := s.provider.Delete(sweepCtx, item.StorageKey); err != nil {
			observability.FromContext(ctx).Warn("media retention sweep object delete failed",
				"error", err, "upload_id", item.UploadID)
			continue // 对象删除失败则保留行，下一轮重试
		}
		deletedObjects++
		if err := s.repo.DeleteUploadRow(sweepCtx, item.UploadID); err != nil {
			observability.FromContext(ctx).Warn("media retention sweep row delete failed",
				"error", err, "upload_id", item.UploadID)
			continue
		}
		deletedRows++
	}
	if deletedRows > 0 || len(stale) > 0 {
		observability.FromContext(ctx).Info("media retention sweep",
			"stale", len(stale), "objects_deleted", deletedObjects, "rows_deleted", deletedRows)
	}
}

// sweepOrphanUploadObjects 清理 uploads/ 前缀下已无会话行引用且超过保留期的
// 孤儿对象（判定与窗口约束见上方孤儿清理常量注释）。逐页列举 → 过滤未过期
// 对象 → 批量查引用 → 删除无引用对象；任一环节失败只告警，下一轮重试。
func (s *Service) sweepOrphanUploadObjects(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(ctx, orphanUploadSweepTimeout)
	defer cancel()
	logger := observability.FromContext(ctx)
	cutoff := time.Now().UTC().AddDate(0, 0, -staleUploadRetentionDays)
	var scanned, candidates, deleted, failed int
	limitReached := false
	token := ""
scanPages:
	for page := 0; page < orphanUploadListMaxPages; page++ {
		listing, err := s.provider.ListObjects(sweepCtx, orphanUploadObjectPrefix, token, orphanUploadListPageSize)
		if err != nil {
			logger.Warn("media retention orphan list failed", "error", err, "scanned", scanned, "deleted", deleted)
			return
		}
		scanned += len(listing.Objects)
		expiredKeys := make([]string, 0, len(listing.Objects))
		for _, object := range listing.Objects {
			if object.Key == "" {
				continue
			}
			if object.LastModified.IsZero() {
				// 无法判断对象年龄：保守跳过（可能是刚上传的对象），告警供排查。
				logger.Warn("media retention orphan object missing last-modified, skipped", "storage_key", object.Key)
				continue
			}
			if !object.LastModified.Before(cutoff) {
				continue
			}
			expiredKeys = append(expiredKeys, object.Key)
		}
		candidates += len(expiredKeys)
		if len(expiredKeys) > 0 {
			referenced, err := s.repo.ExistingStorageKeys(sweepCtx, expiredKeys)
			if err != nil {
				// 引用状态不可确认时不得删除任何对象：可能误删在途上传的对象。
				logger.Warn("media retention orphan reference check failed", "error", err, "candidates", len(expiredKeys))
				return
			}
			for _, storageKey := range expiredKeys {
				if _, ok := referenced[storageKey]; ok {
					continue
				}
				if err := sweepCtx.Err(); err != nil {
					// 单轮预算耗尽（超时/停机）：剩余对象留给下一轮，避免继续向已取消
					// 的 ctx 发删除请求刷告警。
					logger.Warn("media retention orphan sweep budget exhausted",
						"error", err, "scanned", scanned, "deleted", deleted, "failed", failed)
					return
				}
				if deleted >= orphanUploadDeleteLimit {
					limitReached = true
					break scanPages
				}
				if err := s.provider.Delete(sweepCtx, storageKey); err != nil {
					failed++
					logger.Warn("media retention orphan object delete failed", "error", err, "storage_key", storageKey)
					continue
				}
				deleted++
			}
		}
		if !listing.IsTruncated || listing.NextContinuationToken == "" {
			break
		}
		token = listing.NextContinuationToken
	}
	if scanned > 0 || deleted > 0 {
		logger.Info("media retention orphan sweep",
			"scanned", scanned, "candidates", candidates, "objects_deleted", deleted,
			"failed", failed, "delete_limit_reached", limitReached)
	}
}
