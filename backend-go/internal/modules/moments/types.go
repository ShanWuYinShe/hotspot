package moments

import "hotspot/backend-go/internal/platform/types"

type UserInfo = types.UserInfo

type Location struct {
	Point struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"point"`
	LocationName string `json:"location_name"`
}

// Reaction mirrors spec api_spec_types.md Reaction (per-emoji aggregate with
// reacted_by_me). Defined here rather than imported from the reactions module
// so moments does not depend on reactions at the type level; the values are
// filled by a reactions.Service adapter at read time.
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

type MyMomentState struct {
	IsFavorited bool `json:"is_favorited"`
}

type MomentImage struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
}

type MomentItem struct {
	MomentID      string         `json:"moment_id"`
	Description   string         `json:"description"`
	Images        []MomentImage  `json:"images,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Mentions      []string       `json:"mentions,omitempty"`
	CreatedAt     string         `json:"created_at"`
	LikesCount    int            `json:"likes_count"`
	CommentsCount int            `json:"comments_count"`
	SharesCount   int            `json:"shares_count"`
	User          UserInfo       `json:"user"`
	Location      Location       `json:"location"`
	PrivacyLevel  string         `json:"privacy_level,omitempty"`
	// Reactions is the per-emoji aggregate (@auth-only reacted_by_me). Omitted
	// when the moment has no reactions. LikesCount above is the total across
	// all emojis, kept for sort/heat computations and legacy clients.
	Reactions  []Reaction      `json:"reactions,omitempty"`
	MyState    *MyMomentState  `json:"my_state,omitempty"`
}

type CreateMomentRequest struct {
	Description  string   `json:"description"`
	Location     Location `json:"location"`
	PrivacyLevel string   `json:"privacy_level,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	ImageFileIDs []string `json:"image_file_ids,omitempty"`
	Mentions     []string `json:"mentions,omitempty"`
}

type MomentOpenTheDoorRequest struct {
	MomentID string `json:"moment_id"`
}

type PaginatedMoments struct {
	Items      []MomentItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type SearchMomentsRequest struct {
	Keyword        *string           `json:"keyword,omitempty"`
	Tag            *string           `json:"tag,omitempty"`
	Radius         *int              `json:"radius,omitempty"`
	SortBy         string            `json:"sort_by,omitempty"`
	ReferencePoint GeoPoint          `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type GeoPoint = types.GeoPoint

type PaginationParams = types.PaginationParams

type EditMomentRequest struct {
	MomentID     string    `json:"moment_id"`
	Description  *string   `json:"description,omitempty"`
	Location     *Location `json:"location,omitempty"`
	PrivacyLevel *string   `json:"privacy_level,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	ImageFileIDs []string  `json:"image_file_ids,omitempty"`
	Mentions     []string  `json:"mentions,omitempty"`
}

type DeleteMomentRequest struct {
	MomentID string `json:"moment_id"`
}

type FavoriteMomentRequest struct {
	MomentID string `json:"moment_id"`
	Active   bool   `json:"active"`
}

type UnfavoriteMomentRequest struct {
	MomentID string `json:"moment_id"`
}

type MyMomentsRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type MyFavoritesRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}
