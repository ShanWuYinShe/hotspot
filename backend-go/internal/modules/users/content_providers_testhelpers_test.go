package users

import (
	"context"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
)

// stubGroupsProvider / stubMomentsProvider 是 content provider 的最小测试桩：
// groups/moments 内存仓储已迁入各自模块测试构建（P0-2），users 用例只关心
// users 对本模块契约的聚合/过滤行为，不重复构造其它模块实现。
type stubGroupsProvider struct {
	groupsByUser map[string][]grouptypes.GroupItem
	commonGroups map[string]int
	ownedCount   int
	err          error
}

func (s *stubGroupsProvider) ListGroupsByUser(_ context.Context, userID string, limit int, _ string) (grouptypes.PaginatedGroups, error) {
	if s.err != nil {
		return grouptypes.PaginatedGroups{}, s.err
	}
	items := append([]grouptypes.GroupItem(nil), s.groupsByUser[userID]...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return grouptypes.PaginatedGroups{Items: items}, nil
}

func (s *stubGroupsProvider) CountCommonGroups(_ context.Context, viewerUserID, targetUserID string) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.commonGroups[viewerUserID+"|"+targetUserID], nil
}

func (s *stubGroupsProvider) OwnedGroupCount(context.Context, string) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.ownedCount, nil
}

type stubMomentsProvider struct {
	momentsByUser map[string][]momenttypes.MomentItem
	reacted       map[string][]momenttypes.MomentItem
	err           error
}

func (s *stubMomentsProvider) ListMomentsByAuthor(_ context.Context, userID string, limit int, _ string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	if s.err != nil {
		return momenttypes.PaginatedMoments{}, s.err
	}
	items := make([]momenttypes.MomentItem, 0)
	for _, item := range s.momentsByUser[userID] {
		if !includePrivate && item.PrivacyLevel == "private" {
			continue
		}
		items = append(items, item)
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return momenttypes.PaginatedMoments{Items: items}, nil
}

func (s *stubMomentsProvider) ListReactedByUser(_ context.Context, userID, _ string, limit int, _ string) (momenttypes.PaginatedMoments, error) {
	if s.err != nil {
		return momenttypes.PaginatedMoments{}, s.err
	}
	items := append([]momenttypes.MomentItem(nil), s.reacted[userID]...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return momenttypes.PaginatedMoments{Items: items}, nil
}
