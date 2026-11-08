package moments

import (
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

// 以下辅助从生产代码下沉到测试：生产路径已不使用，仅测试替身实现需要。

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

func paginateNearbyMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	decoded, err := decodeFetchCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	limit = types.NormalizeLimit(limit)
	start := findNearbyCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = types.StrPtr(encodeFetchCursor(itemCursor{Score: last.Score, Distance: last.Distance, CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339Nano), ID: last.Item.MomentID}))
	}
	return resp, nil
}

func findNearbyCursorStart(items []momentCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
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

func compareSearchTuple(score float64, at, id string, cur momentSearchCursor) int {
	if score != cur.Score {
		if score < cur.Score {
			return -1
		}
		return 1
	}
	if at != cur.At {
		if at < cur.At {
			return -1
		}
		return 1
	}
	return strings.Compare(id, cur.ID)
}
