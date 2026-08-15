package moments

import (
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
)

const momentSearchCursorNamespace = "moments.search"

type momentSearchCursor struct {
	Sort  string `json:"sort"`
	Score int64  `json:"score"`
	At    string `json:"at"`
	ID    string `json:"id"`
}

func normalizeMomentSearchSort(sortBy string) string {
	switch sortBy {
	case "newest", "popular":
		return sortBy
	default:
		return "relevance"
	}
}

// momentSearchScore is the keyset score used by all three moment search sort
// modes. relevance counts keyword/tag matches; popular counts interactions;
// newest keeps a constant score so the keyset degenerates to
// (created_at, moment_id).
func momentSearchScore(item MomentItem, sortBy, keyword, tag string) int64 {
	switch normalizeMomentSearchSort(sortBy) {
	case "popular":
		return int64(item.LikesCount + item.CommentsCount)
	case "newest":
		return 0
	default:
		score := int64(0)
		if keyword != "" && strings.Contains(item.Description, keyword) {
			score++
		}
		if tag != "" && containsTag(item.Tags, tag) {
			score++
		}
		return score
	}
}

func encodeMomentSearchCursor(sortBy string, item MomentItem, keyword, tag string) string {
	value, err := cursor.Encode(momentSearchCursorNamespace, momentSearchCursor{
		Sort:  normalizeMomentSearchSort(sortBy),
		Score: momentSearchScore(item, sortBy, keyword, tag),
		At:    item.CreatedAt,
		ID:    item.MomentID,
	})
	if err != nil {
		return ""
	}
	return value
}

func decodeMomentSearchCursor(value string) (*momentSearchCursor, error) {
	if value == "" {
		return nil, nil
	}
	var decoded momentSearchCursor
	if err := cursor.Decode(momentSearchCursorNamespace, value, &decoded); err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded.Sort == "" || decoded.At == "" || decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	if _, err := time.Parse(time.RFC3339, decoded.At); err != nil {
		return nil, ErrInvalidCursor
	}
	return &decoded, nil
}
