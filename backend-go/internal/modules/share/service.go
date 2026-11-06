package share

import (
	"context"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

// InteractionCounter 维护瞬间内部互动热度（仅排序用途；契约 shares_count 读时聚合）。
type InteractionCounter interface {
	AdjustInteractionCount(ctx context.Context, momentID string, delta int)
}

type Service struct {
	repo               Repository
	interactionCounter InteractionCounter
}

func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) SetInteractionCounter(c InteractionCounter) { s.interactionCounter = c }

func (s *Service) Create(ctx context.Context, req CreateShareRequest) (ShareCreated, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return ShareCreated{}, ErrInvalidShareRequest
	}
	cmd, err := newCreateShareCommand(principal.UserID, req)
	if err != nil {
		return ShareCreated{}, err
	}
	result, err := s.repo.Create(ctx, cmd)
	if err != nil {
		observability.L().Error("share.create failed", "error", err)
		return ShareCreated{}, err
	}
	// 分享创建后维护内部互动热度（+1）。失败由 moments 侧 WARN，不影响分享主流程；
	// 分享撤销不回减（该列只服务排序启发式，弱一致可接受，契约为读时聚合）。
	if s.interactionCounter != nil && cmd.ShareType == ShareTypeMoment {
		if sharedMomentID, _ := cmd.Payload["shared_moment_id"].(string); sharedMomentID != "" {
			s.interactionCounter.AdjustInteractionCount(ctx, sharedMomentID, 1)
		}
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, req GetShareRequest) (ShareDetail, error) {
	query, err := newGetShareQuery(req)
	if err != nil {
		return ShareDetail{}, err
	}
	item, err := s.repo.Get(ctx, query)
	if err != nil {
		observability.L().Error("share.get failed", "error", err, "share_code", observability.Redact(query.ShareCode))
		return ShareDetail{}, err
	}
	return item, nil
}

// ValidateShareCode 校验 share_code 存在且有效（未撤销/未过期）。
// 供消息模块发送 share 类型消息时调用（spec api_spec_chat.md「消息发送统一规则」：
// 无效返回错误，防发送失效链接）。
func (s *Service) ValidateShareCode(ctx context.Context, shareCode string) error {
	query, err := newGetShareQuery(GetShareRequest{ShareCode: shareCode})
	if err != nil {
		return err
	}
	_, err = s.repo.Get(ctx, query)
	return err
}

func (s *Service) List(ctx context.Context, req ListSharesRequest) (PaginatedShares, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedShares{}, ErrInvalidShareRequest
	}
	query, err := newListSharesQuery(principal.UserID, req)
	if err != nil {
		return PaginatedShares{}, err
	}
	items, err := s.repo.List(ctx, query)
	if err != nil {
		observability.L().Error("share.list failed", "error", err)
		return PaginatedShares{}, err
	}
	return items, nil
}

func (s *Service) Revoke(ctx context.Context, req RevokeShareRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return ErrInvalidShareRequest
	}
	shareID := strings.TrimSpace(req.ShareID)
	shareCode := strings.TrimSpace(req.ShareCode)
	if (shareID == "") == (shareCode == "") {
		// 二选一：必须且只能传 share_id 或 share_code 其中之一
		//（spec api_spec_share.md：创建者可能只保留了复制出的链接码）
		return ErrInvalidShareRequest
	}
	if shareID == "" {
		// 按分享码定位（不过滤 revoked/expired）：撤销已撤销/已过期的分享应
		// 幂等成功，与按 share_id 撤销的口径一致（审查 P2-8 幂等分裂——此前
		// 走 Get，revoked 被映射为不存在 → 返回 404）。
		resolved, found, err := s.repo.ResolveShareIDByCode(ctx, shareCode)
		if err != nil {
			observability.L().Error("share.revoke resolve by code failed", "error", err, "share_code", observability.Redact(shareCode))
			return err
		}
		if !found {
			return ErrShareNotFound
		}
		shareID = resolved
	}
	cmd := revokeShareCommand{CreatorUserID: principal.UserID, ShareID: shareID}
	if err := s.repo.Revoke(ctx, cmd); err != nil {
		observability.L().Error("share.revoke failed", "error", err, "share_id", cmd.ShareID)
		return err
	}
	return nil
}

func newCreateShareCommand(creatorUserID string, req CreateShareRequest) (createShareCommand, error) {
	shareType := normalizeShareType(req.ShareType)
	if creatorUserID == "" || shareType == "" || req.Payload == nil {
		return createShareCommand{}, ErrInvalidShareRequest
	}
	payload := normalizeSharePayload(shareType, req.Payload)
	if payload == nil {
		return createShareCommand{}, ErrInvalidShareRequest
	}
	var expiresAt *time.Time
	if req.ExpiresInSeconds != 0 {
		if req.ExpiresInSeconds <= 0 || req.ExpiresInSeconds > maxShareExpiresInSec {
			return createShareCommand{}, ErrInvalidShareRequest
		}
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
		expiresAt = &t
	}
	return createShareCommand{CreatorUserID: creatorUserID, ShareType: shareType, Payload: payload, ExpiresAt: expiresAt}, nil
}

func newGetShareQuery(req GetShareRequest) (getShareQuery, error) {
	shareCode := normalizeShareCode(req.ShareCode)
	if !validShareCode(shareCode) {
		return getShareQuery{}, ErrInvalidShareRequest
	}
	return getShareQuery{ShareCode: shareCode}, nil
}

func newListSharesQuery(creatorUserID string, req ListSharesRequest) (listSharesQuery, error) {
	if creatorUserID == "" {
		return listSharesQuery{}, ErrInvalidShareRequest
	}
	shareType := ""
	if strings.TrimSpace(req.ShareType) != "" {
		shareType = normalizeShareType(req.ShareType)
		if shareType == "" {
			return listSharesQuery{}, ErrInvalidShareRequest
		}
	}
	limit, cursor := normalizeSharePagination(req.Pagination)
	if _, err := decodeShareCursor(cursor); err != nil {
		return listSharesQuery{}, err
	}
	return listSharesQuery{CreatorUserID: creatorUserID, ShareType: shareType, Limit: limit, Cursor: cursor}, nil
}

func normalizeSharePayload(shareType string, payload map[string]any) map[string]any {
	switch shareType {
	case ShareTypeUser:
		id, ok := payload["shared_user_id"].(string)
		if !ok || strings.TrimSpace(id) == "" || len(payload) != 1 {
			return nil
		}
		return map[string]any{"shared_user_id": strings.TrimSpace(id)}
	case ShareTypeGroup:
		id, ok := payload["shared_group_id"].(string)
		if !ok || strings.TrimSpace(id) == "" || len(payload) != 1 {
			return nil
		}
		return map[string]any{"shared_group_id": strings.TrimSpace(id)}
	case ShareTypeMoment:
		id, ok := payload["shared_moment_id"].(string)
		if !ok || strings.TrimSpace(id) == "" || len(payload) != 1 {
			return nil
		}
		return map[string]any{"shared_moment_id": strings.TrimSpace(id)}
	case ShareTypeLocation:
		sharedLocation, ok := payload["shared_location"].(map[string]any)
		if !ok || len(payload) != 1 {
			return nil
		}
		point, ok := sharedLocation["point"].(map[string]any)
		if !ok {
			return nil
		}
		lat, latOK := point["latitude"].(float64)
		lng, lngOK := point["longitude"].(float64)
		locationName, nameOK := sharedLocation["location_name"].(string)
		if !latOK || !lngOK || !nameOK {
			return nil
		}
		locationName = strings.TrimSpace(locationName)
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 || locationName == "" || len(locationName) > 100 || len(sharedLocation) != 2 || len(point) != 2 {
			return nil
		}
		return map[string]any{
			"shared_location": map[string]any{
				"point": map[string]any{
					"latitude":  lat,
					"longitude": lng,
				},
				"location_name": locationName,
			},
		}
	default:
		return nil
	}
}
