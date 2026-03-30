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

func (r *PostgresRepository) Get(ctx context.Context, query getShareQuery) (ShareDetail, error) {
	queryable, ok, err := r.queryable()
	if err != nil {
		return ShareDetail{}, err
	}
	if !ok {
		return r.fallback.Get(ctx, query)
	}
	var record shareRecord
	var payloadJSON string
	var creator ShareUserInfo
	if err := queryable.QueryRow(ctx, queryGetShare, query.ShareCode).Scan(
		&record.ShareID,
		&record.ShareCode,
		&record.ShareType,
		&record.CreatorUserID,
		&creator.Nickname,
		&creator.AvatarURL,
		&payloadJSON,
		&record.CreatedAt,
		&record.ExpiresAt,
		&record.IsRevoked,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShareDetail{}, ErrShareNotFound
		}
		return ShareDetail{}, err
	}
	creator.UserID = record.CreatorUserID
	if err := json.Unmarshal([]byte(payloadJSON), &record.Payload); err != nil {
		return ShareDetail{}, err
	}
	if shareUnavailable(record) {
		return ShareDetail{}, ErrShareNotFound
	}
	preview, err := resolveSharePreview(ctx, queryable, record.ShareType, record.Payload)
	if err != nil && !errors.Is(err, ErrShareNotFound) {
		return ShareDetail{}, err
	}
	if errors.Is(err, ErrShareNotFound) {
		preview = nil
	}
	return ShareDetail{
		Share: ShareBase{
			ShareID:   record.ShareID,
			ShareCode: record.ShareCode,
			ShareType: record.ShareType,
			Creator:   creator,
			CreatedAt: record.CreatedAt,
			ExpiresAt: record.ExpiresAt,
		},
		Payload: record.Payload,
		Preview: preview,
	}, nil
}

func (r *PostgresRepository) List(ctx context.Context, query listSharesQuery) (PaginatedShares, error) {
	queryable, ok, err := r.queryable()
	if err != nil {
		return PaginatedShares{}, err
	}
	if !ok {
		return r.fallback.List(ctx, query)
	}
	cursor, err := decodeShareCursor(query.Cursor)
	if err != nil {
		return PaginatedShares{}, err
	}
	cursorCreatedAt := ""
	cursorShareID := ""
	if cursor != nil {
		cursorCreatedAt = cursor.CreatedAt
		cursorShareID = cursor.ShareID
	}
	rows, err := queryable.Query(ctx, queryListShares, query.CreatorUserID, query.ShareType, cursorCreatedAt, cursorShareID, query.Limit+1)
	if err != nil {
		return PaginatedShares{}, err
	}
	defer rows.Close()
	items := make([]ShareListItem, 0, query.Limit+1)
	for rows.Next() {
		var item ShareListItem
		var payloadJSON string
		if err := rows.Scan(&item.ShareID, &item.ShareCode, &item.ShareType, &item.CreatedAt, &item.ExpiresAt, &item.IsRevoked, &payloadJSON); err != nil {
			return PaginatedShares{}, err
		}
		if payloadJSON != "" {
			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				return PaginatedShares{}, err
			}
			preview, err := resolveSharePreview(ctx, queryable, item.ShareType, payload)
			if err == nil {
				item.Preview = preview
			} else if !errors.Is(err, ErrShareNotFound) {
				return PaginatedShares{}, err
			}
		}
		if item.ExpiresAt != "" && isExpired(item.ExpiresAt) {
			item.IsRevoked = true
		}
		if item.IsRevoked {
			item.Preview = nil
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedShares{}, err
	}
	page := PaginatedShares{}
	if len(items) > query.Limit {
		last := items[query.Limit-1]
		page.NextCursor = encodeShareCursor(shareCursor{CreatedAt: last.CreatedAt, ShareID: last.ShareID})
		items = items[:query.Limit]
	}
	page.Items = items
	return page, nil
}

func resolveSharePreview(ctx context.Context, queryable persistence.QueryableDB, shareType string, payload map[string]any) (*SharePreview, error) {
	switch shareType {
	case ShareTypeUser:
		_, nickname, avatarURL, err := loadSharedUser(ctx, queryable, getPayloadID(payload, "shared_user_id"))
		if err != nil {
			return nil, err
		}
		return &SharePreview{Title: nickname, Subtitle: "用户", CoverURL: avatarURL}, nil
	case ShareTypeGroup:
		_, name, description, avatarURL, err := loadSharedGroup(ctx, queryable, getPayloadID(payload, "shared_group_id"))
		if err != nil {
			return nil, err
		}
		return &SharePreview{Title: name, Subtitle: description, CoverURL: avatarURL}, nil
	case ShareTypeMoment:
		_, description, locationName, authorNickname, authorAvatarURL, err := loadSharedMoment(ctx, queryable, getPayloadID(payload, "shared_moment_id"))
		if err != nil {
			return nil, err
		}
		title := strings.TrimSpace(description)
		if title == "" {
			title = locationName
		}
		return &SharePreview{Title: title, Subtitle: authorNickname, CoverURL: authorAvatarURL}, nil
	case ShareTypeLocation:
		sharedLocation, _ := payload["shared_location"].(map[string]any)
		locationName, _ := sharedLocation["location_name"].(string)
		return &SharePreview{Title: locationName, Subtitle: "位置"}, nil
	default:
		return nil, ErrInvalidShareRequest
	}
}

func shareUnavailable(record shareRecord) bool {
	if record.IsRevoked {
		return true
	}
	return isExpired(record.ExpiresAt)
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
