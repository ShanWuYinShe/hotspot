package groups

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

const (
	defaultSearchRadius = 5000
	minSearchRadius     = 100
	maxSearchRadius     = 50000
	defaultSearchLimit  = 20
	maxSearchLimit      = 100

	searchSortRelevance = "relevance"
	searchSortDistance  = "distance"
	searchSortNewest    = "newest"
	searchSortPopular   = "popular"
)

type groupSearchCursor struct {
	SortBy       string `json:"sort_by"`
	GroupID      string `json:"group_id"`
	Distance     int    `json:"distance,omitempty"`
	MembersCount int    `json:"members_count,omitempty"`
	NameMatch    int    `json:"name_match,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type groupSearchMeta struct {
	Distance  int
	Bearing   int
	NameMatch int
	CreatedAt time.Time
}

func hasSearchCriteria(req SearchGroupsRequest) bool {
	return strings.TrimSpace(req.Keyword) != "" || strings.TrimSpace(req.Tag) != "" || req.Radius != nil
}

func normalizedSearchSort(sortBy string) string {
	switch strings.TrimSpace(sortBy) {
	case "", searchSortRelevance:
		return searchSortRelevance
	case searchSortDistance:
		return searchSortDistance
	case searchSortNewest:
		return searchSortNewest
	case searchSortPopular:
		return searchSortPopular
	default:
		return ""
	}
}

func normalizedSearchRequest(req SearchGroupsRequest) SearchGroupsRequest {
	req.Keyword = strings.TrimSpace(req.Keyword)
	req.Tag = strings.TrimSpace(req.Tag)
	req.SortBy = normalizedSearchSort(req.SortBy)
	if req.SortBy == "" {
		req.SortBy = searchSortRelevance
	}
	if req.Radius == nil && (req.Keyword != "" || req.Tag != "") {
		radius := defaultSearchRadius
		req.Radius = &radius
	}
	return req
}

func searchGeoPoint(point GeoPoint) geo.Point {
	return geo.Point{Latitude: point.Latitude, Longitude: point.Longitude}
}

func groupLocationPoint(location RelativeLocation) geo.Point {
	return geo.Point{Latitude: location.Point.Latitude, Longitude: location.Point.Longitude}
}

func searchNameMatchScore(item GroupItem, keyword string) int {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return 0
	}
	name := strings.ToLower(item.Group.Name)
	description := strings.ToLower(item.Description)
	switch {
	case name == keyword:
		return 3
	case strings.Contains(name, keyword):
		return 2
	case strings.Contains(description, keyword):
		return 1
	default:
		return 0
	}
}

func groupCreatedAt(item GroupItem) time.Time {
	if item.CreatedAt == nil || *item.CreatedAt == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, *item.CreatedAt)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func matchesSearchCursor(meta groupSearchMeta, item GroupItem, cursor groupSearchCursor) bool {
	switch cursor.SortBy {
	case searchSortDistance:
		if meta.Distance != cursor.Distance {
			return meta.Distance > cursor.Distance
		}
		return item.Group.GroupID > cursor.GroupID
	case searchSortNewest:
		cursorTime, _ := time.Parse(time.RFC3339, cursor.CreatedAt)
		if !meta.CreatedAt.Equal(cursorTime) {
			return meta.CreatedAt.Before(cursorTime)
		}
		return item.Group.GroupID > cursor.GroupID
	case searchSortPopular:
		if item.Group.MembersCount != cursor.MembersCount {
			return item.Group.MembersCount < cursor.MembersCount
		}
		return item.Group.GroupID > cursor.GroupID
	default:
		if meta.NameMatch != cursor.NameMatch {
			return meta.NameMatch < cursor.NameMatch
		}
		if item.Group.MembersCount != cursor.MembersCount {
			return item.Group.MembersCount < cursor.MembersCount
		}
		return item.Group.GroupID > cursor.GroupID
	}
}

func encodeSearchCursor(cursor groupSearchCursor) (string, error) {
	if cursor.GroupID == "" {
		return "", nil
	}
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeSearchCursor(raw string) (groupSearchCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return groupSearchCursor{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return groupSearchCursor{}, fmt.Errorf("decode search cursor: %w", err)
	}
	var cursor groupSearchCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return groupSearchCursor{}, fmt.Errorf("unmarshal search cursor: %w", err)
	}
	return cursor, nil
}

func sortGroupsForSearch(items []GroupItem, metas map[string]groupSearchMeta, sortBy string) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		leftMeta := metas[left.Group.GroupID]
		rightMeta := metas[right.Group.GroupID]
		switch sortBy {
		case searchSortDistance:
			if leftMeta.Distance != rightMeta.Distance {
				return leftMeta.Distance < rightMeta.Distance
			}
		case searchSortNewest:
			if !leftMeta.CreatedAt.Equal(rightMeta.CreatedAt) {
				return leftMeta.CreatedAt.After(rightMeta.CreatedAt)
			}
		case searchSortPopular:
			if left.Group.MembersCount != right.Group.MembersCount {
				return left.Group.MembersCount > right.Group.MembersCount
			}
		default:
			if leftMeta.NameMatch != rightMeta.NameMatch {
				return leftMeta.NameMatch > rightMeta.NameMatch
			}
			if left.Group.MembersCount != right.Group.MembersCount {
				return left.Group.MembersCount > right.Group.MembersCount
			}
		}
		return left.Group.GroupID < right.Group.GroupID
	})
}

// searchCursorForGroupItem encodes the sort position of a group search result:
// the fields that pin its place under the requested ordering. Cursors are
// opaque to clients; they only re-locate the item inside the freshly
// recomputed, Go-sorted result list.
func searchCursorForGroupItem(item GroupItem, meta groupSearchMeta, sortBy string) groupSearchCursor {
	return groupSearchCursor{
		SortBy:       sortBy,
		GroupID:      item.Group.GroupID,
		Distance:     meta.Distance,
		MembersCount: item.Group.MembersCount,
		NameMatch:    meta.NameMatch,
		CreatedAt:    groupCreatedAt(item).Format(time.RFC3339),
	}
}

// paginateSearchGroups slices a fully sorted result list into pages. It
// locates the cursor's item in the list and resumes right after it; if that
// item no longer exists (e.g. the group was dismissed or its membership count
// changed between pages), it falls back to skipping the rows that sort after
// the cursor.
func paginateSearchGroups(items []GroupItem, metas map[string]groupSearchMeta, sortBy string, pagination *PaginationParams) (PaginatedGroups, error) {
	limit := paginationLimit(pagination, defaultSearchLimit, maxSearchLimit)
	cursor, err := decodeSearchCursor(paginationCursor(pagination))
	if err != nil {
		return PaginatedGroups{}, err
	}
	if cursor.GroupID != "" {
		start := -1
		for i, item := range items {
			if searchCursorForGroupItem(item, metas[item.Group.GroupID], sortBy) == cursor {
				start = i + 1
				break
			}
		}
		if start < 0 {
			filtered := make([]GroupItem, 0, len(items))
			for _, item := range items {
				if matchesSearchCursor(metas[item.Group.GroupID], item, cursor) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		} else {
			items = items[start:]
		}
	}
	response := PaginatedGroups{}
	if len(items) <= limit {
		response.Items = items
		return response, nil
	}
	response.Items = items[:limit]
	last := response.Items[len(response.Items)-1]
	next, err := encodeSearchCursor(searchCursorForGroupItem(last, metas[last.Group.GroupID], sortBy))
	if err != nil {
		return PaginatedGroups{}, err
	}
	response.NextCursor = next
	return response, nil
}
