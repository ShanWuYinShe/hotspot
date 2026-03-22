package share

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, shareCode string) (ShareDetail, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Get(ctx, shareCode)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return ShareDetail{}, false, persistence.ErrNotImplemented
	}
	var shareID, code, shareType, creatorUserID, payloadJSON, createdAt, expiresAt string
	var payload map[string]any
	var isRevoked bool
	if err := queryable.QueryRow(ctx, queryGetShare, strings.ToUpper(shareCode)).Scan(&shareID, &code, &shareType, &creatorUserID, &payloadJSON, &createdAt, &expiresAt, &isRevoked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShareDetail{}, false, nil
		}
		return ShareDetail{}, false, err
	}
	if payloadJSON != "" {
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return ShareDetail{}, false, err
		}
	}
	if isRevoked || isExpired(expiresAt) {
		return ShareDetail{}, false, nil
	}
	share := map[string]any{"share_id": shareID, "share_code": code, "share_type": shareType, "creator": map[string]any{"user_id": creatorUserID, "nickname": creatorUserID}, "created_at": createdAt}
	if expiresAt != "" {
		share["expires_at"] = expiresAt
	}
	return ShareDetail{Share: share, Payload: payload, Preview: map[string]any{"title": shareType}}, true, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, userID, shareID string) (ShareListItem, bool, error) {
	list, err := r.List(ctx, userID)
	if err != nil {
		return ShareListItem{}, false, err
	}
	for _, item := range list.Items {
		if item.ShareID == shareID {
			return item, true, nil
		}
	}
	return ShareListItem{}, false, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID string) (PaginatedShares, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.List(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedShares{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListShares, userID)
	if err != nil {
		return PaginatedShares{}, err
	}
	defer rows.Close()
	items := []ShareListItem{}
	for rows.Next() {
		var item ShareListItem
		if err := rows.Scan(&item.ShareID, &item.ShareCode, &item.ShareType, &item.CreatedAt, &item.ExpiresAt, &item.IsRevoked); err != nil {
			return PaginatedShares{}, err
		}
		if item.ExpiresAt != "" && isExpired(item.ExpiresAt) {
			item.IsRevoked = true
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedShares{}, err
	}
	return PaginatedShares{Items: items}, nil
}

func isExpired(expiresAt string) bool {
	if expiresAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return false
	}
	return !t.After(time.Now().UTC())
}
