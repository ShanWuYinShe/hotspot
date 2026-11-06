package admin

import (
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/worker"
)

// 保留期清理：审计日志（ADMIN_AUDIT_RETENTION_DAYS，默认 365 天）与终态举报
// （REPORT_RETENTION_DAYS，默认 365 天）。两类表只增不减是持久化审计 P1-9 的
// 明确缺口；删除为后台幂等操作，不参与在线请求路径。
//
// 多实例部署注意：每实例独立启动且启动即先跑一轮（与既有 notify/media/auth
// sweeper 一致），滚动发布时会出现短时重复删除；后续统一周期任务框架再引入
// jitter + 分布式锁（见设计文档 Batch E）。
var (
	retentionSweepMu       sync.Mutex
	retentionSweepStarted  bool
	retentionSweepInterval = 24 * time.Hour
	retentionSweepTimeout  = 5 * time.Minute
)

// StartRetentionSweeper 启动审计日志/终态举报保留期清理；两条目天数均 <=0 时
// 不启动。goroutine 绑定 ctx 随优雅停机退出；重复调用 no-op。
func (s *Service) StartRetentionSweeper(ctx context.Context, auditRetentionDays, reportRetentionDays int) {
	if auditRetentionDays <= 0 && reportRetentionDays <= 0 {
		return
	}
	retentionSweepMu.Lock()
	defer retentionSweepMu.Unlock()
	if retentionSweepStarted {
		return
	}
	retentionSweepStarted = true
	worker.GoRestartable("admin.retention_sweeper", func() {
		defer func() {
			retentionSweepMu.Lock()
			retentionSweepStarted = false
			retentionSweepMu.Unlock()
		}()
		s.pruneRetentionOnce(ctx, auditRetentionDays, reportRetentionDays)
		ticker := time.NewTicker(retentionSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.pruneRetentionOnce(ctx, auditRetentionDays, reportRetentionDays)
			}
		}
	})
}

// pruneRetentionOnce 执行一轮清理；单轮独立超时，避免一轮慢查询拖住后续周期。
func (s *Service) pruneRetentionOnce(ctx context.Context, auditRetentionDays, reportRetentionDays int) {
	sweepCtx, cancel := context.WithTimeout(ctx, retentionSweepTimeout)
	defer cancel()
	now := time.Now().UTC()
	if auditRetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -auditRetentionDays)
		deleted, err := s.repo.DeleteAuditLogsBefore(sweepCtx, cutoff)
		if err != nil {
			observability.L().Warn("admin.retention audit prune failed", "error", err)
		} else if deleted > 0 {
			observability.L().Info("admin.retention audit pruned", "deleted", deleted, "cutoff", cutoff)
		}
	}
	if reportRetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -reportRetentionDays)
		deleted, err := s.repo.DeleteHandledReportsBefore(sweepCtx, cutoff)
		if err != nil {
			observability.L().Warn("admin.retention report prune failed", "error", err)
		} else if deleted > 0 {
			observability.L().Info("admin.retention reports pruned", "deleted", deleted, "cutoff", cutoff)
		}
	}
}
