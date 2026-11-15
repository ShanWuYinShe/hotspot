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

var (
	retentionSweepMu      sync.Mutex
	retentionSweepStarted bool
)

// StartRetentionSweeper 启动上传会话保留期清理：超过保留期的未完成会话，
// 先删其可能已部分上传的存储对象（幂等，404 视为已删），再删会话行。
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
	s.sweepStaleUploads(ctx) // 启动即先跑一轮
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepStaleUploads(ctx)
		}
	}
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
