package moments

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/types"
)

// SetLocationProvider 接入 users 模块的位置解析器（reference_point 回退链第 2 级，
// spec api_spec_types.md）。可选：nil 时跳过用户最近位置一级。
func (s *Service) SetLocationProvider(lp LocationProvider) { s.locations = lp }

// FetchFollowing 关注动态：我关注的人发布的公开瞬间（spec moment.md →
// moments/fetch_following，原 feed/following 迁移）。按 created_at DESC,
// moment_id DESC keyset 强一致分页；无匿名版（依赖查看者关注关系）。
func (s *Service) FetchFollowing(ctx context.Context, req FetchFollowingMomentsRequest) (PaginatedMoments, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedMoments{}, ErrInvalidFetchRequest
	}
	refPt := s.resolveReferencePoint(ctx, req.ReferencePoint)
	query := momentsQuery{
		ViewerUserID: principal.UserID,
		// 参考点按回退链解析（spec api_spec_types.md）；解析失败（显式传入非法坐标）
		// 时不填充响应侧 distance/bearing_degrees。
		HasReferencePoint: refPt != nil,
		Limit:             normalizeFetchLimit(req.Pagination),
		Cursor:            types.PaginationCursor(req.Pagination),
	}
	if refPt != nil {
		query.ReferencePoint = *refPt
	}
	items, err := s.repo.ListFollowingMoments(ctx, query)
	if err != nil {
		observability.L().Error("moments.fetch_following failed", "error", err)
		return PaginatedMoments{}, err
	}
	s.hydrateImagesBatch(ctx, items.Items)
	s.fillReactionsBatch(ctx, items.Items, principal.UserID)
	s.fillAuthorRelations(ctx, items.Items, principal.UserID)
	ensureMomentArraysBatch(items.Items)
	return items, nil
}

// FetchNearby 附近瞬间（spec moment.md → moments/fetch_nearby，原
// feed/nearby_moments 迁移）。distance 主序、radius 缺省 5000 受
// search_radius_max 裁剪、reference_point 按回退链解析；匿名版无查看者
// 字段（is_favorited 等返回默认值）。
func (s *Service) FetchNearby(ctx context.Context, req FetchNearbyMomentsRequest) (PaginatedMoments, error) {
	viewerID := ""
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok {
		viewerID = principal.UserID
	}
	refPt := s.resolveReferencePoint(ctx, req.ReferencePoint)
	if refPt == nil {
		return PaginatedMoments{}, ErrInvalidFetchRequest
	}
	query := momentsQuery{
		ViewerUserID:   viewerID,
		ReferencePoint: *refPt,
		// 附近流参考点恒已解析（未解析直接报参数错误），相对位置字段恒填充。
		HasReferencePoint: true,
		Radius:            s.normalizeRadius(ctx, req.Radius),
		Limit:             normalizeFetchLimit(req.Pagination),
		Cursor:            types.PaginationCursor(req.Pagination),
	}
	items, err := s.repo.ListNearbyMoments(ctx, query)
	if err != nil {
		observability.L().Error("moments.fetch_nearby failed", "error", err)
		return PaginatedMoments{}, err
	}
	s.hydrateImagesBatch(ctx, items.Items)
	s.fillReactionsBatch(ctx, items.Items, viewerID)
	s.fillAuthorRelations(ctx, items.Items, viewerID)
	ensureMomentArraysBatch(items.Items)
	return items, nil
}

// ensureMomentArraysBatch 批量保证恒定数组字段序列化为 [] 而非 null（spec 恒定形状）。
func ensureMomentArraysBatch(items []MomentItem) {
	for i := range items {
		ensureMomentArrays(&items[i])
	}
}
