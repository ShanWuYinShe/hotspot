package comments

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type MyCommentState struct {
	IsLiked bool `json:"is_liked"`
}

type CommentItem struct {
	CommentID    string          `json:"comment_id"`
	User         UserInfo        `json:"user"`
	Content      string          `json:"content"`
	ParentID     string          `json:"parent_id,omitempty"`
	Mentions     []string        `json:"mentions,omitempty"`
	CreatedAt    string          `json:"created_at"`
	LikesCount   int             `json:"likes_count"`
	RepliesCount int             `json:"replies_count"`
	MyState      *MyCommentState `json:"my_state,omitempty"`
}

type CreateCommentRequest struct {
	MomentID string   `json:"moment_id"`
	Content  string   `json:"content"`
	ParentID string   `json:"parent_id,omitempty"`
	Mentions []string `json:"mentions,omitempty"`
}

type ListCommentsRequest struct {
	MomentID string `json:"moment_id"`
}

type PaginatedComments struct {
	Items      []CommentItem `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}
