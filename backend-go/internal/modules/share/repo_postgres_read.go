package share

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

// 列表页 preview 批量查询：按目标类型集合一次取回，替代逐条 resolveSharePreview
// 的 N+1（一页 N 条分享 = N 次目标表查询）。目标可见性过滤与单条 query*Exists
// 完全一致（已注销用户/已删群组/非公开或已删瞬间按不存在处理）。
const (
	queryShareUsersByIDs = `
SELECT user_id, nickname, COALESCE(avatar_url, '')
FROM users
WHERE user_id = ANY($1::text[])
  AND is_deleted = FALSE`

	queryShareGroupsByIDs = `
SELECT group_id, name, COALESCE(description, ''), COALESCE(avatar_url, '')
FROM groups
WHERE group_id = ANY($1::text[])
  AND deleted_at IS NULL`

	queryShareMomentsByIDs = `
SELECT m.moment_id, m.description, m.location_name, u.nickname, COALESCE(u.avatar_url, '')
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.moment_id = ANY($1::text[])
  AND m.privacy_level = 'public'
  AND m.is_deleted = FALSE`
)

func (r *PostgresRepository) Get(ctx context.Context, query getShareQuery) (ShareDetail, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return ShareDetail{}, err
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
			ExpiresAt: types.StrPtr(record.ExpiresAt),
		},
		Payload: record.Payload,
		Preview: preview,
	}, nil
}

func (r *PostgresRepository) List(ctx context.Context, query listSharesQuery) (PaginatedShares, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return PaginatedShares{}, err
	}
	cursor, err := decodeShareCursor(query.Cursor)
	if err != nil {
		return PaginatedShares{}, err
	}
	var cursorCreatedAt any
	cursorShareID := ""
	if cursor != nil {
		cursorCreatedAt = cursor.CreatedAt
		cursorShareID = cursor.ShareID
	}
	var shareType any = query.ShareType
	if query.ShareType == "" {
		shareType = nil
	}
	rows, err := queryable.Query(ctx, queryListShares, query.CreatorUserID, shareType, cursorCreatedAt, cursorShareID, query.Limit+1)
	if err != nil {
		return PaginatedShares{}, err
	}
	defer rows.Close()
	items := make([]ShareListItem, 0, query.Limit+1)
	rawCreatedAt := make([]time.Time, 0, query.Limit+1)
	// payloads 与 items 按位对应（空 payload 为 nil，不参与 preview 解析）。
	payloads := make([]map[string]any, 0, query.Limit+1)
	for rows.Next() {
		var item ShareListItem
		var payloadJSON string
		var createdAtRaw time.Time
		if err := rows.Scan(&item.ShareID, &item.ShareCode, &item.ShareType, &item.CreatedAt, &item.ExpiresAt, &item.IsRevoked, &payloadJSON, &createdAtRaw); err != nil {
			return PaginatedShares{}, err
		}
		rawCreatedAt = append(rawCreatedAt, createdAtRaw.UTC())
		var payload map[string]any
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				return PaginatedShares{}, err
			}
		}
		payloads = append(payloads, payload)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedShares{}, err
	}
	// preview 批量解析：按目标类型集合各查一次（此前在行循环里逐条
	// resolveSharePreview，一页 N 条分享 = N 次目标表查询，N+1 问题）。
	targets := make([]sharePreviewTarget, 0, len(items))
	targetIndex := make([]int, 0, len(items))
	for i := range items {
		if payloads[i] == nil {
			continue
		}
		targets = append(targets, sharePreviewTarget{shareType: items[i].ShareType, payload: payloads[i]})
		targetIndex = append(targetIndex, i)
	}
	if len(targets) > 0 {
		previews, err := resolveSharePreviews(ctx, queryable, targets)
		if err != nil {
			return PaginatedShares{}, err
		}
		for j, index := range targetIndex {
			items[index].Preview = previews[j]
		}
	}
	// 过期与撤销统一呈现为不可用（is_revoked=true、preview=nil），与单条路径一致。
	for i := range items {
		if items[i].ExpiresAt != nil && *items[i].ExpiresAt != "" && isExpired(*items[i].ExpiresAt) {
			items[i].IsRevoked = true
		}
		if items[i].IsRevoked {
			items[i].Preview = nil
		}
	}
	page := PaginatedShares{}
	if len(items) > query.Limit {
		last := items[query.Limit-1]
		// 游标用行内真实时间（微秒）：CreatedAt 是契约展示字段（秒），同秒分享
		// 用它做 keyset 会翻页重复/丢行。
		c, err := encodeShareCursor(shareCursor{CreatedAt: rawCreatedAt[query.Limit-1].Format(time.RFC3339Nano), ShareID: last.ShareID})
		if err != nil {
			return PaginatedShares{}, err
		}
		page.NextCursor = &c
		items = items[:query.Limit]
	}
	page.Items = items
	return page, nil
}

// sharePreviewTarget 是列表页待解析 preview 的目标（分享类型 + payload）。
type sharePreviewTarget struct {
	shareType string
	payload   map[string]any
}

// sharedUserPreview/sharedGroupPreview/sharedMomentPreview 是批量查询的行结果，
// 字段与单条 loadShared* 的返回值一一对应。
type sharedUserPreview struct {
	nickname  string
	avatarURL string
}

type sharedGroupPreview struct {
	name        string
	description string
	avatarURL   string
}

type sharedMomentPreview struct {
	description     string
	locationName    string
	authorNickname  string
	authorAvatarURL string
}

// resolveSharePreviews 批量解析列表页 preview（每类目标一次查询），返回与
// targets 按位对应的 preview。目标不存在/不可见时对应元素为 nil——与单条
// resolveSharePreview 的 ErrShareNotFound 降级语义完全一致（列表保留该行、
// preview 置空）；未知分享类型与单条路径一致整体返回 ErrInvalidShareRequest。
func resolveSharePreviews(ctx context.Context, queryable persistence.Queryable, targets []sharePreviewTarget) ([]*SharePreview, error) {
	userIDs := make(map[string]struct{})
	groupIDs := make(map[string]struct{})
	momentIDs := make(map[string]struct{})
	for _, target := range targets {
		switch target.shareType {
		case ShareTypeUser:
			if id := getPayloadID(target.payload, "shared_user_id"); id != "" {
				userIDs[id] = struct{}{}
			}
		case ShareTypeGroup:
			if id := getPayloadID(target.payload, "shared_group_id"); id != "" {
				groupIDs[id] = struct{}{}
			}
		case ShareTypeMoment:
			if id := getPayloadID(target.payload, "shared_moment_id"); id != "" {
				momentIDs[id] = struct{}{}
			}
		case ShareTypeLocation:
		default:
			return nil, ErrInvalidShareRequest
		}
	}
	users, err := loadSharedUsersByIDs(ctx, queryable, sortedIDSet(userIDs))
	if err != nil {
		return nil, err
	}
	groups, err := loadSharedGroupsByIDs(ctx, queryable, sortedIDSet(groupIDs))
	if err != nil {
		return nil, err
	}
	moments, err := loadSharedMomentsByIDs(ctx, queryable, sortedIDSet(momentIDs))
	if err != nil {
		return nil, err
	}
	previews := make([]*SharePreview, len(targets))
	for i, target := range targets {
		switch target.shareType {
		case ShareTypeUser:
			user, ok := users[getPayloadID(target.payload, "shared_user_id")]
			if !ok {
				continue
			}
			previews[i] = &SharePreview{Title: user.nickname, Subtitle: types.StrPtr("用户"), CoverURL: types.StrPtr(user.avatarURL)}
		case ShareTypeGroup:
			group, ok := groups[getPayloadID(target.payload, "shared_group_id")]
			if !ok {
				continue
			}
			previews[i] = &SharePreview{Title: group.name, Subtitle: types.StrPtr(group.description), CoverURL: types.StrPtr(group.avatarURL)}
		case ShareTypeMoment:
			moment, ok := moments[getPayloadID(target.payload, "shared_moment_id")]
			if !ok {
				continue
			}
			title := strings.TrimSpace(moment.description)
			if title == "" {
				title = moment.locationName
			}
			previews[i] = &SharePreview{Title: title, Subtitle: types.StrPtr(moment.authorNickname), CoverURL: types.StrPtr(moment.authorAvatarURL)}
		case ShareTypeLocation:
			sharedLocation, _ := target.payload["shared_location"].(map[string]any)
			locationName, _ := sharedLocation["location_name"].(string)
			previews[i] = &SharePreview{Title: locationName, Subtitle: types.StrPtr("位置")}
		}
	}
	return previews, nil
}

func loadSharedUsersByIDs(ctx context.Context, queryable persistence.Queryable, ids []string) (map[string]sharedUserPreview, error) {
	out := make(map[string]sharedUserPreview, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := queryable.Query(ctx, queryShareUsersByIDs, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var item sharedUserPreview
		if err := rows.Scan(&id, &item.nickname, &item.avatarURL); err != nil {
			return nil, err
		}
		out[id] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func loadSharedGroupsByIDs(ctx context.Context, queryable persistence.Queryable, ids []string) (map[string]sharedGroupPreview, error) {
	out := make(map[string]sharedGroupPreview, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := queryable.Query(ctx, queryShareGroupsByIDs, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var item sharedGroupPreview
		if err := rows.Scan(&id, &item.name, &item.description, &item.avatarURL); err != nil {
			return nil, err
		}
		out[id] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func loadSharedMomentsByIDs(ctx context.Context, queryable persistence.Queryable, ids []string) (map[string]sharedMomentPreview, error) {
	out := make(map[string]sharedMomentPreview, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := queryable.Query(ctx, queryShareMomentsByIDs, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var item sharedMomentPreview
		if err := rows.Scan(&id, &item.description, &item.locationName, &item.authorNickname, &item.authorAvatarURL); err != nil {
			return nil, err
		}
		out[id] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// sortedIDSet 把集合展开为有序切片（批量查询参数稳定，便于日志与测试比对）。
func sortedIDSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func resolveSharePreview(ctx context.Context, queryable persistence.Queryable, shareType string, payload map[string]any) (*SharePreview, error) {
	switch shareType {
	case ShareTypeUser:
		_, nickname, avatarURL, err := loadSharedUser(ctx, queryable, getPayloadID(payload, "shared_user_id"))
		if err != nil {
			return nil, err
		}
		return &SharePreview{Title: nickname, Subtitle: types.StrPtr("用户"), CoverURL: types.StrPtr(avatarURL)}, nil
	case ShareTypeGroup:
		_, name, description, avatarURL, err := loadSharedGroup(ctx, queryable, getPayloadID(payload, "shared_group_id"))
		if err != nil {
			return nil, err
		}
		return &SharePreview{Title: name, Subtitle: types.StrPtr(description), CoverURL: types.StrPtr(avatarURL)}, nil
	case ShareTypeMoment:
		_, description, locationName, authorNickname, authorAvatarURL, err := loadSharedMoment(ctx, queryable, getPayloadID(payload, "shared_moment_id"))
		if err != nil {
			return nil, err
		}
		title := strings.TrimSpace(description)
		if title == "" {
			title = locationName
		}
		return &SharePreview{Title: title, Subtitle: types.StrPtr(authorNickname), CoverURL: types.StrPtr(authorAvatarURL)}, nil
	case ShareTypeLocation:
		sharedLocation, _ := payload["shared_location"].(map[string]any)
		locationName, _ := sharedLocation["location_name"].(string)
		return &SharePreview{Title: locationName, Subtitle: types.StrPtr("位置")}, nil
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
	t, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return false
	}
	return !t.After(time.Now().UTC())
}
