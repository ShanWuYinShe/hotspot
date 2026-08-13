package feed

import (
	"time"

	platformcursor "hotspot/backend-go/internal/platform/cursor"
)

func paginateUsers(items []userCandidate, limit int, cursor string) (PaginatedUsers, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedUsers{}, err
	}
	start := findUserCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedUsers{Items: make([]UserItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeFeedCursor(itemCursor{Distance: items[end-1].Distance, ID: items[end-1].Item.User.UserID})
	}
	return resp, nil
}

func paginateGroups(items []groupCandidate, limit int, cursor string) (PaginatedGroups, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedGroups{}, err
	}
	start := findGroupCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedGroups{Items: make([]GroupItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeFeedCursor(itemCursor{Distance: items[end-1].Distance, ID: items[end-1].Item.Group.GroupID})
	}
	return resp, nil
}

func paginateFollowingMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	start := findFollowingCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = encodeFeedCursor(itemCursor{CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID})
	}
	return resp, nil
}

func paginateNearbyMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	start := findNearbyCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = encodeFeedCursor(itemCursor{Score: last.Score, Distance: last.Distance, CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID})
	}
	return resp, nil
}

// compareUserCandidates 排序规则：距离升序 → 共同好友数降序 → userID 降序
// （ID 大小写只作稳定 tie-break，保证分页键集序与游标一致）。
func compareUserCandidates(left, right userCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Item.SocialContext != nil && right.Item.SocialContext != nil && left.Item.SocialContext.MutualFriendsCount != right.Item.SocialContext.MutualFriendsCount {
		return right.Item.SocialContext.MutualFriendsCount - left.Item.SocialContext.MutualFriendsCount
	}
	if left.Item.User.UserID > right.Item.User.UserID {
		return -1
	}
	if left.Item.User.UserID < right.Item.User.UserID {
		return 1
	}
	return 0
}

// compareGroupCandidates 排序规则：距离升序 → 群内好友数降序 → groupID 降序。
func compareGroupCandidates(left, right groupCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Item.FriendsInGroup != right.Item.FriendsInGroup {
		return right.Item.FriendsInGroup - left.Item.FriendsInGroup
	}
	if left.Item.Group.GroupID > right.Item.Group.GroupID {
		return -1
	}
	if left.Item.Group.GroupID < right.Item.Group.GroupID {
		return 1
	}
	return 0
}

// compareFollowingCandidates 排序规则：发布时间降序（最新在前）→ momentID 降序。
func compareFollowingCandidates(left, right momentCandidate) int {
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

// compareNearbyCandidates 排序规则（spec feed.md → nearby 排序主序）：
// 距离升序为主序（服务端计算 distance）；热度评分仅作为次级信号——实现为
// 距离差在近似阈值内时优先热度，否则严格按距离。简化实现：距离升序 → 热度降序
// → 发布时间降序 → momentID 降序。
func compareNearbyCandidates(left, right momentCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Score != right.Score {
		return right.Score - left.Score
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

func findUserCursorStart(items []userCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	for idx, item := range items {
		cmp := compareUserCandidates(item, userCandidate{Item: UserItem{User: UserInfo{UserID: cursor.ID}}, Distance: cursor.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findGroupCursorStart(items []groupCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	for idx, item := range items {
		cmp := compareGroupCandidates(item, groupCandidate{Item: GroupItem{Group: GroupInfo{GroupID: cursor.ID}}, Distance: cursor.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findFollowingCursorStart(items []momentCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339, cursor.CreatedAt)
	for idx, item := range items {
		cmp := compareFollowingCandidates(item, momentCandidate{Item: MomentItem{MomentID: cursor.ID}, CreatedAt: parsedAt.UTC()})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findNearbyCursorStart(items []momentCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339, cursor.CreatedAt)
	for idx, item := range items {
		cmp := compareNearbyCandidates(item, momentCandidate{Item: MomentItem{MomentID: cursor.ID}, Score: cursor.Score, Distance: cursor.Distance, CreatedAt: parsedAt.UTC()})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

var feedCursorCodec = platformcursor.NewCodec("feed")

func encodeFeedCursor(cursor itemCursor) string {
	token, err := feedCursorCodec.Encode(cursor)
	if err != nil {
		return ""
	}
	return token
}

func decodeFeedCursor(cursor string) (*itemCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	var decoded itemCursor
	if err := feedCursorCodec.Decode(cursor, &decoded); err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	if decoded.CreatedAt != "" {
		if _, err := time.Parse(time.RFC3339, decoded.CreatedAt); err != nil {
			return nil, ErrInvalidCursor
		}
	}
	return &decoded, nil
}

var _ Repository = (*PostgresRepository)(nil)
