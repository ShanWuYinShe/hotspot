package auth

import (
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/worker"
)

// 数据保留策略（成熟度审计：auth_sessions 含 IP PII 无限累积、验证码行
// 含验证码明文永不删除）。模式对齐 notify StartPruneSweeper：模块级后台
// goroutine，周期执行 PG 侧保留期清理；删除幂等，失败静默等下一轮。
const (
	retentionSweepCadence         = time.Hour
	authSessionRetentionDays      = 30 // 已撤销会话保留 30 天（排障窗口）后清除
	verificationCodeRetentionDays = 7  // 验证码/重置码过期 7 天后清除（行含验证码明文）
)

var (
	retentionSweepMu      sync.Mutex
	retentionSweepStarted bool
)

// StartRetentionSweeper 启动 PG 数据保留期清理（auth_sessions / 注册验证码 /
// 密码重置请求）。幂等：重复调用为 no-op；goroutine 绑定 ctx 随优雅停机退出。
func (m *Module) StartRetentionSweeper(ctx context.Context) {
	retentionSweepMu.Lock()
	defer retentionSweepMu.Unlock()
	if retentionSweepStarted {
		return
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	retentionSweepStarted = true
	worker.GoRestartable("auth.retention_sweeper", func() {
		defer cancel()
		defer func() {
			retentionSweepMu.Lock()
			retentionSweepStarted = false
			retentionSweepMu.Unlock()
		}()
		ticker := time.NewTicker(retentionSweepCadence)
		defer ticker.Stop()
		m.sweepRetention(sweepCtx) // 启动即先跑一轮
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.sweepRetention(sweepCtx)
			}
		}
	})
}

func (m *Module) sweepRetention(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	now := time.Now().UTC()
	deleted, err := m.repo.SweepRetention(sweepCtx, now,
		authSessionRetentionDays*24*time.Hour, verificationCodeRetentionDays*24*time.Hour)
	if err != nil {
		observability.L().Warn("auth retention sweep failed", "error", err)
		return
	}
	if deleted > 0 {
		observability.L().Info("auth retention sweep", "deleted", deleted)
	}
}
