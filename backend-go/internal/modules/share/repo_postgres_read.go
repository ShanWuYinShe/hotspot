package share

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) Get(ctx context.Context, shareCode string) (ShareDetail, bool) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.Get(ctx, shareCode)
	}
	var shareID, code, shareType, creatorUserID, createdAt string
	var isRevoked bool
	if err := queryable.QueryRow(ctx, queryGetShare, shareCode).Scan(&shareID, &code, &shareType, &creatorUserID, &createdAt, &isRevoked); err != nil {
		return r.fallback.Get(ctx, shareCode)
	}
	if isRevoked {
		return ShareDetail{}, false
	}
	return ShareDetail{Share: map[string]any{"share_id": shareID, "share_code": code, "share_type": shareType, "creator": map[string]any{"user_id": creatorUserID, "nickname": creatorUserID}, "created_at": createdAt}, Payload: map[string]any{}, Preview: map[string]any{"title": shareType}}, true
}
func (r *PostgresRepository) List(ctx context.Context, userID string) PaginatedShares {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.List(ctx, userID)
	}
	rows, err := queryable.Query(ctx, queryListShares, userID)
	if err != nil {
		return r.fallback.List(ctx, userID)
	}
	defer rows.Close()
	items := []ShareListItem{}
	for rows.Next() {
		var item ShareListItem
		if err := rows.Scan(&item.ShareID, &item.ShareCode, &item.ShareType, &item.CreatedAt, &item.IsRevoked); err != nil {
			return r.fallback.List(ctx, userID)
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.fallback.List(ctx, userID)
	}
	return PaginatedShares{Items: items}
}
