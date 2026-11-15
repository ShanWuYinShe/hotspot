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

// MomentCoverResolver 把 moment 封面的对象存储 key 签名成绝对读 URL；media.Service
// 已满足（PresignGet），由 bootstrap 注入。允许为空：未注入时保留头像兜底。
type MomentCoverResolver interface {
	PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error)
}

// shareMomentCoverTTL 分享封面（瞬间首图）presign 读 URL 有效期，取 7 天与
// moments 模块的 momentReadTTL 一致：分享链可能被转发到群聊/朋友圈后很久才被
// 打开，签名过短会让图片 403；而落地页/OG 爬虫每次请求都会重新解析并签名，
// 过期只影响已下发缓存的旧 URL，不会永久失效。
const shareMomentCoverTTL = 7 * 24 * time.Hour

type Service struct {
	repo               Repository
	interactionCounter InteractionCounter
	coverResolver      MomentCoverResolver
}

func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) SetInteractionCounter(c InteractionCounter) { s.interactionCounter = c }

// SetCoverResolver 注入 moment 分享封面签名器（bootstrap 传 media.Service）。
func (s *Service) SetCoverResolver(resolver MomentCoverResolver) { s.coverResolver = resolver }

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
		observability.FromContext(ctx).Error("share.create failed", "error", err)
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
		observability.FromContext(ctx).Error("share.get failed", "error", err, "share_code", observability.Redact(query.ShareCode))
		return ShareDetail{}, err
	}
	s.applyMomentCover(ctx, item.Preview)
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
		observability.FromContext(ctx).Error("share.list failed", "error", err)
		return PaginatedShares{}, err
	}
	// 列表页与单条入口共用同一封面口径（「我的分享」列表也展示封面）。
	for i := range items.Items {
		s.applyMomentCover(ctx, items.Items[i].Preview)
	}
	return items, nil
}

// applyMomentCover 把 preview 上由仓储带出的 moment 封面 storage key 签名成
// 绝对读 URL 并覆盖 CoverURL（产品决策：moment 分享封面用瞬间第一张图片）。
// 纯文字瞬间（coverStorageKey 为空）保留仓储填写的作者头像兜底；未注入
// resolver 或签名失败时同样保留头像兜底并 WARN——封面只是展示增强，不能让
// 分享解析/OG 落地页失败（预览爬虫会即时抓取，失败即回退而不是重试）。
func (s *Service) applyMomentCover(ctx context.Context, preview *SharePreview) {
	if preview == nil || preview.coverStorageKey == "" {
		return
	}
	if s.coverResolver == nil {
		observability.FromContext(ctx).Warn("share.moment_cover resolver not configured, fallback to avatar", "storage_key", preview.coverStorageKey)
		return
	}
	url, err := s.coverResolver.PresignGet(ctx, preview.coverStorageKey, shareMomentCoverTTL)
	if err != nil {
		observability.FromContext(ctx).Warn("share.moment_cover presign failed, fallback to avatar", "error", err, "storage_key", preview.coverStorageKey)
		return
	}
	preview.CoverURL = &url
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
			observability.FromContext(ctx).Error("share.revoke resolve by code failed", "error", err, "share_code", observability.Redact(shareCode))
			return err
		}
		if !found {
			return ErrShareNotFound
		}
		shareID = resolved
	}
	cmd := revokeShareCommand{CreatorUserID: principal.UserID, ShareID: shareID}
	if err := s.repo.Revoke(ctx, cmd); err != nil {
		observability.FromContext(ctx).Error("share.revoke failed", "error", err, "share_id", cmd.ShareID)
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
