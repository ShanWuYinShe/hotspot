package notify

import (
	"context"
	"encoding/json"

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
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q, true
	}
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	q, ok := r.factory.DB.(persistence.QueryableDB)
	return q, ok
}

func (r *PostgresRepository) Append(ctx context.Context, userID string, item NotifyItem) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		// 与单条 Append 的降级语义一致：显式报告未实现，而非静默丢弃公告。
		return persistence.ErrNotImplemented
	}
	if _, ok := r.factory.DB.(persistence.QueryableDB); !ok {
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

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, ids []string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkNotifysRead, userID, ids)
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllNotifysRead, userID)
}
