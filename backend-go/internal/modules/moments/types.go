package moments

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type Location struct {
	Point struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"point"`
	LocationName string `json:"location_name"`
}

type MyMomentState struct {
	IsLiked     bool `json:"is_liked"`
	IsFavorited bool `json:"is_favorited"`
}

type MomentItem struct {
	MomentID      string         `json:"moment_id"`
	Description   string         `json:"description"`
	Tags          []string       `json:"tags,omitempty"`
	Mentions      []string       `json:"mentions,omitempty"`
	CreatedAt     string         `json:"created_at"`
	LikesCount    int            `json:"likes_count"`
	CommentsCount int            `json:"comments_count"`
	SharesCount   int            `json:"shares_count"`
	User          UserInfo       `json:"user"`
	Location      Location       `json:"location"`
	PrivacyLevel  string         `json:"privacy_level,omitempty"`
	MyState       *MyMomentState `json:"my_state,omitempty"`
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
	Keyword        *string                `json:"keyword,omitempty"`
	Tag            *string                `json:"tag,omitempty"`
	Radius         *int                   `json:"radius,omitempty"`
	SortBy         string                 `json:"sort_by,omitempty"`
	ReferencePoint GeoPoint               `json:"reference_point"`
	Pagination     *PaginationParams      `json:"pagination,omitempty"`
}

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type EditMomentRequest struct {
	MomentID      string   `json:"moment_id"`
	Description   *string  `json:"description,omitempty"`
	Location      *Location `json:"location,omitempty"`
	PrivacyLevel  *string  `json:"privacy_level,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	ImageFileIDs  []string `json:"image_file_ids,omitempty"`
	Mentions      []string `json:"mentions,omitempty"`
}

type DeleteMomentRequest struct {
	MomentID string `json:"moment_id"`
}
