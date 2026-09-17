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
	retentionSweepCadence    = time.Hour
	staleUploadRetentionDays = 7   // 未完成上传会话保留 7 天
	staleUploadBatchLimit    = 100 // 单轮最多处理条数，防止单轮时长失控
)

var (
	retentionSweepMu      sync.Mutex
	retentionSweepStarted bool
)

// StartRetentionSweeper 启动上传会话保留期清理：超过保留期的未完成会话，
// 先删其可能已部分上传的存储对象（幂等，404 视为已删），再删会话行。
// goroutine 绑定 ctx 随优雅停机退出；重复调用 no-op。
func (s *Service) StartRetentionSweeper(ctx context.Context) {
	retentionSweepMu.Lock()
	defer retentionSweepMu.Unlock()
	if retentionSweepStarted {
		return
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	retentionSweepStarted = true
	worker.GoRestartable("media.retention_sweeper", func() {
		defer cancel()
		defer func() {
			retentionSweepMu.Lock()
			retentionSweepStarted = false
			retentionSweepMu.Unlock()
		}()
		ticker := time.NewTicker(retentionSweepCadence)
		defer ticker.Stop()
		s.sweepStaleUploads(sweepCtx) // 启动即先跑一轮
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sweepStaleUploads(sweepCtx)
			}
		}
	})
}

func (s *Service) sweepStaleUploads(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cutoff := time.Now().UTC().AddDate(0, 0, -staleUploadRetentionDays)
	stale, err := s.repo.ListStaleUploads(sweepCtx, cutoff, staleUploadBatchLimit)
	if err != nil {
		observability.L().Warn("media retention sweep list failed", "error", err)
		return
	}
	var deletedObjects, deletedRows int
	for _, item := range stale {
		// 对象可能从未真正上传（presign 后放弃）——404 视为已删，不算失败。
		if err := s.provider.Delete(sweepCtx, item.StorageKey); err != nil {
			observability.L().Warn("media retention sweep object delete failed",
				"error", err, "upload_id", item.UploadID)
			continue // 对象删除失败则保留行，下一轮重试
		}
		deletedObjects++
		if err := s.repo.DeleteUploadRow(sweepCtx, item.UploadID); err != nil {
			observability.L().Warn("media retention sweep row delete failed",
				"error", err, "upload_id", item.UploadID)
			continue
		}
		deletedRows++
	}
	if deletedRows > 0 || len(stale) > 0 {
		observability.L().Info("media retention sweep",
			"stale", len(stale), "objects_deleted", deletedObjects, "rows_deleted", deletedRows)
	}
}
