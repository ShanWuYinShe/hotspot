package users

import (
	"sort"
	"strings"
)

// 搜索内存实现与 PG 实现共用的纯函数/候选类型（关键词、半径、排序、游标分页
// 口径保持一致；内存仓储实现已迁入测试构建，这些纯函数必须留在生产代码，
// 架构审计 P0-2）。

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func normalizeSearchRadius(radius *int) int {
	value := 5000
	if radius != nil {
		value = *radius
	}
	if value < 100 {
		value = 100
	}
	if value > 50000 {
		value = 50000
	}
	return value
}

func normalizeSearchSort(sortBy string) string {
	switch sortBy {
	case "distance", "newest":
		return sortBy
	default:
		return "relevance"
	}
}

func sortSearchCandidates(items []searchProfileCandidate, sortBy string) {
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		switch sortBy {
		case "distance":
			if left.Distance != right.Distance {
				return left.Distance < right.Distance
			}
			if left.RelevanceScore != right.RelevanceScore {
				return left.RelevanceScore > right.RelevanceScore
			}
			if left.Profile.LastOnline != right.Profile.LastOnline {
				return left.Profile.LastOnline > right.Profile.LastOnline
			}
		case "newest":
			if left.Profile.CreatedAt != right.Profile.CreatedAt {
				return left.Profile.CreatedAt > right.Profile.CreatedAt
			}
		default:
			if left.RelevanceScore != right.RelevanceScore {
				return left.RelevanceScore > right.RelevanceScore
			}
			if left.Profile.LastOnline != right.Profile.LastOnline {
				return left.Profile.LastOnline > right.Profile.LastOnline
			}
		}
		return left.Profile.UserID < right.Profile.UserID
	})
}

func searchRelevanceScore(profile MyProfile, keyword, tag string) int {
	score := 0
	lowerNickname := strings.ToLower(profile.Nickname)
	lowerBio := strings.ToLower(profile.Bio)
	if keyword != "" {
		switch {
		case lowerNickname == keyword:
			score += 6
		case strings.HasPrefix(lowerNickname, keyword):
			score += 4
		case strings.Contains(lowerNickname, keyword):
			score += 3
		}
		if strings.Contains(lowerBio, keyword) {
			score += 2
		}
	}
	if tag != "" && containsFold(profile.Tags, tag) {
		score += 3
	}
	return score
}

func paginateSearchCandidates(items []searchProfileCandidate, p *PaginationParams) PaginatedProfiles {
	limit, cursor := normalizePagination(p)
	start := 0
	if cursor != "" {
		decoded, err := decodeSearchCursor(cursor)
		if err != nil {
			return PaginatedProfiles{InvalidCursor: true}
		}
		matched := false
		for i, item := range items {
			if searchCursorForItem(item) == decoded {
				start = i + 1
				matched = true
				break
			}
		}
		if !matched {
			return PaginatedProfiles{InvalidCursor: true}
		}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	result := PaginatedProfiles{Items: make([]MyProfile, 0, maxInt(end-start, 0))}
	for _, item := range items[start:end] {
		result.Items = append(result.Items, item.Profile)
	}
	if end < len(items) && end > start {
		result.NextCursor = encodeSearchCursor(items[end-1])
	}
	return result
}

type searchProfileCandidate struct {
	Profile        MyProfile
	Distance       int
	BearingDegrees int
	SortBy         string
	RelevanceScore int
}

type searchCursor struct {
	SortBy     string `json:"sort_by"`
	UserID     string `json:"user_id"`
	Distance   int    `json:"distance,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	LastOnline string `json:"last_online,omitempty"`
	// RelevanceScore 仅 relevance 排序的 SQL keyset 分页使用（旧候选路径不依赖）。
	RelevanceScore int `json:"relevance_score,omitempty"`
}

func searchCursorForItem(item searchProfileCandidate) searchCursor {
	return searchCursor{SortBy: item.SortBy, UserID: item.Profile.UserID, Distance: item.Distance, CreatedAt: item.Profile.CreatedAt, LastOnline: item.Profile.LastOnline, RelevanceScore: item.RelevanceScore}
}

func encodeSearchCursor(item searchProfileCandidate) string {
	token, err := usersCursorCodec.Encode(searchCursorForItem(item))
	if err != nil {
		return ""
	}
	return token
}

func decodeSearchCursor(raw string) (searchCursor, error) {
	var decoded searchCursor
	if err := usersCursorCodec.Decode(raw, &decoded); err != nil {
		return searchCursor{}, ErrInvalidCursor
	}
	if decoded.SortBy == "" || decoded.UserID == "" {
		return searchCursor{}, ErrInvalidCursor
	}
	return decoded, nil
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
