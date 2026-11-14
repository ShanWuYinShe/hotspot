package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory       *persistence.Factory
	announcements AnnouncementNotificationSource
	// txRunner 仅供单测注入「模拟回滚」的事务执行器替身；生产为 nil，
	// runInTx 默认走 persistence.RunInTransaction。
	txRunner persistence.TxRunner
}

// SetAnnouncementNotificationSource 注入 system 模块的公告虚拟通知能力
// （bootstrap 装配）。未注入时通知列表现有行为不变。
func (r *PostgresRepository) SetAnnouncementNotificationSource(source AnnouncementNotificationSource) {
	r.announcements = source
}

// SetTxRunner 注入事务执行器（单测用回滚替身模拟失败撤销）。生产不注入：
// 默认行为仍是 persistence.RunInTransaction，两者语义一致。
func (r *PostgresRepository) SetTxRunner(runner persistence.TxRunner) {
	r.txRunner = runner
}

// runInTx 在同一事务内执行 fn：无注入时用 persistence.RunInTransaction 开启
// pgx 事务并经 ctx 传递，system 模块的公告已读仓储（同一 factory）因此加入
// 同一事务，公告已读与普通通知已读要么同提交要么同回滚。
func (r *PostgresRepository) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if r.txRunner != nil {
		return r.txRunner(ctx, fn)
	}
	return persistence.RunInTransaction(ctx, r.factory, fn)
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

// txQueryable returns the transaction-bound Queryable when one is in effect
// (real pgx tx via RunInTransaction), otherwise falls back to the shared pool.
// The fallback matters for tests, where RunInTransaction detects a stub backend
// and runs fn without injecting a tx.
func (r *PostgresRepository) txQueryable(ctx context.Context) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, r.factory)
}

func (r *PostgresRepository) Append(ctx context.Context, userID string, item NotifyItem) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	item = normalizeNotifyItem(item)
	payloadJSON, err := json.Marshal(item.Payload)
	if err != nil {
		return err
	}
	return queryable.Exec(ctx, queryInsertNotify, item.NotifyID, userID, item.Type, item.IsRead, item.CreatedAt, string(payloadJSON))
}

func (r *PostgresRepository) AppendBatchPerUser(ctx context.Context, userIDs []string, items []NotifyItem) error {
	// 与单条 Append 的降级语义一致：显式报告未实现，而非静默丢弃通知。
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	if len(userIDs) != len(items) {
		return fmt.Errorf("notify.AppendBatchPerUser: userIDs/items 长度不一致 (%d/%d)", len(userIDs), len(items))
	}
	ids := make([]string, 0, len(items))
	uids := make([]string, 0, len(items))
	typs := make([]string, 0, len(items))
	reads := make([]bool, 0, len(items))
	cats := make([]string, 0, len(items))
	payloads := make([]string, 0, len(items))
	for i, item := range items {
		if userIDs[i] == "" {
			continue
		}
		item = normalizeNotifyItem(item)
		payloadJSON, err := json.Marshal(item.Payload)
		if err != nil {
			return err
		}
		ids = append(ids, item.NotifyID)
		uids = append(uids, userIDs[i])
		typs = append(typs, item.Type)
		reads = append(reads, item.IsRead)
		cats = append(cats, item.CreatedAt)
		payloads = append(payloads, string(payloadJSON))
	}
	if len(ids) == 0 {
		return nil
	}
	// 单事务单往返：此前逐人循环中途失败会留下部分落库（部分用户收到通知、
	// 部分没有），批量后要么全部落库要么整体失败。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		return txQueryable.Exec(txCtx, queryInsertNotifysBatchPerUser, ids, uids, typs, reads, cats, payloads)
	})
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, ids []string) error {
	// 公告虚拟通知的 notify_id（sysann_ 前缀）路由到 system 模块的公告已读表；
	// 其余 ID 走 notifications 表。两类可同时出现（mark_read scope=ids），
	// 两段写必须同事务：中途失败只完成其中一段会把通知中心留在「部分已读」的
	// 分裂状态（公告已读/普通通知未读，或反之），且客户端无法自愈。
	persisted := ids
	var announcementIDs []string
	if r.announcements != nil {
		persisted = make([]string, 0, len(ids))
		for _, id := range ids {
			if announcementID, ok := systemmodule.ParseAnnouncementNotifyID(id); ok {
				announcementIDs = append(announcementIDs, announcementID)
				continue
			}
			persisted = append(persisted, id)
		}
	}
	return r.runInTx(ctx, func(txCtx context.Context) error {
		if len(announcementIDs) > 0 {
			if err := r.announcements.MarkAnnouncementRead(txCtx, userID, announcementIDs); err != nil {
				return err
			}
		}
		// 仅公告 ID 且已启用公告路由时无需访问 notifications 表（保持原有短路）。
		if r.announcements != nil && len(persisted) == 0 {
			return nil
		}
		queryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		return queryable.Exec(txCtx, queryMarkNotifysRead, userID, persisted)
	})
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	// 与 MarkAsRead 同因：公告全部已读 + 普通通知全部已读必须同事务。
	return r.runInTx(ctx, func(txCtx context.Context) error {
		queryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if r.announcements != nil {
			if err := r.announcements.MarkAllAnnouncementsRead(txCtx, userID); err != nil {
				return err
			}
		}
		return queryable.Exec(txCtx, queryMarkAllNotifysRead, userID)
	})
}

// pruneBatchSize 单批删除上限（审查 M-4）：存量千万级已读通知时单条无界
// DELETE 会形成长事务持锁 + WAL 尖刺，分批循环把单事务体量封顶。
const pruneBatchSize = 5000

// PruneReadBefore 物理删除已读且创建早于 before 的通知（repo.go 接口注释）。
// 分批循环执行，直至某批删除数不足一批（删完）。
func (r *PostgresRepository) PruneReadBefore(ctx context.Context, before time.Time) (int64, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var pruned int64
	for {
		rows, err := queryable.Query(ctx, queryPruneReadNotifysBatch, before.UTC(), pruneBatchSize)
		if err != nil {
			return pruned, err
		}
		var batch int64
		for rows.Next() {
			batch++
		}
		if err := rows.Err(); err != nil {
			return pruned, err
		}
		rows.Close()
		pruned += batch
		if batch < pruneBatchSize {
			return pruned, nil
		}
	}
}
