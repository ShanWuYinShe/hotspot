package share

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

type inMemoryShare struct {
	shareRecord
	Creator ShareUserInfo
}

type InMemoryRepository struct {
	mu     sync.RWMutex
	shares map[string]inMemoryShare
	byCode map[string]string
	byUser map[string][]string
	// seq 保证同纳秒内连续创建的 ID 不碰撞（UnixNano 时间戳在快速循环下会重复）。
	seq uint64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{shares: make(map[string]inMemoryShare), byCode: make(map[string]string), byUser: make(map[string][]string)}
}

func (r *InMemoryRepository) Create(_ context.Context, cmd createShareCommand) (ShareCreated, error) {
	if err := ensureMemoryShareTargetExists(cmd.ShareType, cmd.Payload); err != nil {
		return ShareCreated{}, err
	}
	now := time.Now().UTC()
	r.seq++
	shareID := fmt.Sprintf("share_%d_%d", now.UnixNano(), r.seq)
	record := inMemoryShare{
		shareRecord: shareRecord{
			ShareID:       shareID,
			ShareType:     cmd.ShareType,
			CreatorUserID: cmd.CreatorUserID,
			Payload:       cmd.Payload,
			CreatedAt:     now.Format(time.RFC3339),
		},
		Creator: ShareUserInfo{UserID: cmd.CreatorUserID, Nickname: cmd.CreatorUserID},
	}
	if cmd.ExpiresAt != nil {
		record.ExpiresAt = cmd.ExpiresAt.UTC().Format(time.RFC3339)
	}
	for {
		shareCode, err := newMemoryShareCode()
		if err != nil {
			return ShareCreated{}, err
		}
		r.mu.Lock()
		if _, exists := r.byCode[shareCode]; exists {
			r.mu.Unlock()
			continue
		}
		record.ShareCode = shareCode
		r.shares[shareID] = record
		r.byCode[shareCode] = shareID
		r.byUser[cmd.CreatorUserID] = append(r.byUser[cmd.CreatorUserID], shareID)
		r.mu.Unlock()
		return ShareCreated{ShareID: shareID, ShareCode: shareCode, ExpiresAt: record.ExpiresAt}, nil
	}
}

func (r *InMemoryRepository) Get(_ context.Context, query getShareQuery) (ShareDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	shareID, ok := r.byCode[query.ShareCode]
	if !ok {
		return ShareDetail{}, ErrShareNotFound
	}
	record := r.shares[shareID]
	if shareUnavailable(record.shareRecord) {
		return ShareDetail{}, ErrShareNotFound
	}
	preview, err := buildMemorySharePreview(record.ShareType, record.Payload)
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
			Creator:   record.Creator,
			CreatedAt: record.CreatedAt,
			ExpiresAt: record.ExpiresAt,
		},
		Payload: record.Payload,
		Preview: preview,
	}, nil
}

func (r *InMemoryRepository) List(_ context.Context, query listSharesQuery) (PaginatedShares, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := append([]string(nil), r.byUser[query.CreatorUserID]...)
	sort.Slice(ids, func(i, j int) bool {
		left := r.shares[ids[i]]
		right := r.shares[ids[j]]
		if left.CreatedAt == right.CreatedAt {
			return left.ShareID > right.ShareID
		}
		return left.CreatedAt > right.CreatedAt
	})
	cursor, err := decodeShareCursor(query.Cursor)
	if err != nil {
		return PaginatedShares{}, err
	}
	items := make([]ShareListItem, 0, query.Limit+1)
	started := cursor == nil
	for _, id := range ids {
		record := r.shares[id]
		if query.ShareType != "" && record.ShareType != query.ShareType {
			continue
		}
		if !started {
			if record.CreatedAt < cursor.CreatedAt || (record.CreatedAt == cursor.CreatedAt && record.ShareID < cursor.ShareID) {
				started = true
			} else {
				continue
			}
		}
		preview, err := buildMemorySharePreview(record.ShareType, record.Payload)
		if err != nil && !errors.Is(err, ErrShareNotFound) {
			return PaginatedShares{}, err
		}
		item := ShareListItem{ShareID: record.ShareID, ShareCode: record.ShareCode, ShareType: record.ShareType, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt, IsRevoked: record.IsRevoked || isExpired(record.ExpiresAt), Preview: preview}
		if item.IsRevoked {
			item.Preview = nil
		}
		items = append(items, item)
		if len(items) > query.Limit {
			break
		}
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

func (r *InMemoryRepository) Revoke(_ context.Context, cmd revokeShareCommand) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.shares[cmd.ShareID]
	if !ok {
		return ErrShareNotFound
	}
	if record.CreatorUserID != cmd.CreatorUserID {
		return ErrSharePermission
	}
	if record.IsRevoked {
		return nil
	}
	record.IsRevoked = true
	r.shares[cmd.ShareID] = record
	return nil
}

func ensureMemoryShareTargetExists(shareType string, payload map[string]any) error {
	switch shareType {
	case ShareTypeUser:
		if getPayloadID(payload, "shared_user_id") == "" {
			return ErrShareNotFound
		}
	case ShareTypeGroup:
		if getPayloadID(payload, "shared_group_id") == "" {
			return ErrShareNotFound
		}
	case ShareTypeMoment:
		if getPayloadID(payload, "shared_moment_id") == "" {
			return ErrShareNotFound
		}
	case ShareTypeLocation:
		return nil
	default:
		return ErrInvalidShareRequest
	}
	return nil
}

func buildMemorySharePreview(shareType string, payload map[string]any) (*SharePreview, error) {
	switch shareType {
	case ShareTypeUser:
		id := getPayloadID(payload, "shared_user_id")
		if id == "" {
			return nil, ErrShareNotFound
		}
		return &SharePreview{Title: id, Subtitle: "用户"}, nil
	case ShareTypeGroup:
		id := getPayloadID(payload, "shared_group_id")
		if id == "" {
			return nil, ErrShareNotFound
		}
		return &SharePreview{Title: id, Subtitle: "群组"}, nil
	case ShareTypeMoment:
		id := getPayloadID(payload, "shared_moment_id")
		if id == "" {
			return nil, ErrShareNotFound
		}
		return &SharePreview{Title: id, Subtitle: "瞬间"}, nil
	case ShareTypeLocation:
		sharedLocation, _ := payload["shared_location"].(map[string]any)
		locationName, _ := sharedLocation["location_name"].(string)
		if locationName == "" {
			return nil, ErrShareNotFound
		}
		return &SharePreview{Title: locationName, Subtitle: "位置"}, nil
	default:
		return nil, ErrInvalidShareRequest
	}
}

func newMemoryShareCode() (string, error) {
	buf := make([]byte, shareCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := make([]byte, shareCodeLength)
	for i := range buf {
		code[i] = shareCodeAlphabet[int(buf[i])%len(shareCodeAlphabet)]
	}
	return string(code), nil
}
