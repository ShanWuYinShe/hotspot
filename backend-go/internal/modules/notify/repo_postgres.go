package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
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

func (r *PostgresRepository) AppendMany(ctx context.Context, userIDs []string, item NotifyItem) error {
	// 与单条 Append 的降级语义一致：显式报告未实现，而非静默丢弃公告。
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	// Filter empty ids up front so the batch insert doesn't waste a row.
	targets := make([]string, 0, len(userIDs))
	for _, uid := range userIDs {
		if uid != "" {
			targets = append(targets, uid)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	item = normalizeNotifyItem(item)
	payloadJSON, err := json.Marshal(item.Payload)
	if err != nil {
		return err
	}
	// Insert all rows in a single transaction (and a single round trip). The
	// previous per-user loop would commit partial fan-out on a mid-loop
	// failure, leaving some members notified and others not. Each row gets a
	// distinct notify_id derived from the shared id prefix, since the
	// table's PK is notify_id alone and a shared id would collide.
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		return txQueryable.Exec(txCtx, queryInsertNotifysBatch, targets, item.NotifyID, item.Type, item.IsRead, item.CreatedAt, string(payloadJSON))
	})
}

// AppendBatchPerUser 一次往返落库多用户各自独立的通知（@ 提及场景：每个
// 用户独立 notify_id 与 payload，落库结果与逐人 Append 一致）。userIDs 与
// items 按位对应、长度必须相等；空 user_id 跳过（与 AppendMany 一致）。
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkNotifysRead, userID, ids)
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllNotifysRead, userID)
}

// PruneReadBefore 物理删除已读且创建早于 before 的通知（repo.go 接口注释）。
func (r *PostgresRepository) PruneReadBefore(ctx context.Context, before time.Time) (int64, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryPruneReadNotifys, before.UTC())
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var pruned int64
	for rows.Next() {
		pruned++
	}
	return pruned, rows.Err()
}
