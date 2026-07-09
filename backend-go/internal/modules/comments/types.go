package comments

import "hotspot/backend-go/internal/platform/types"

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
	MomentID     string          `json:"-"`
	User         UserInfo        `json:"user"`
	Content      string          `json:"content"`
	ParentID     string          `json:"parent_id,omitempty"`
	Mentions     []string        `json:"mentions,omitempty"`
	CreatedAt    string          `json:"created_at"`
	LikesCount   int             `json:"likes_count"`
	RepliesCount int             `json:"replies_count"`
	IsDeleted    bool            `json:"is_deleted"`
	// IsLiked is the spec-defined top-level @auth-only field (api_spec_moment.md
	// CommentItem.is_liked). It mirrors MyState.IsLiked so clients reading
	// comment.is_liked per the contract get the value. Kept as a pointer so it
	// omits when nil (unauthenticated / no state), matching the `?` optional.
	IsLiked     *bool            `json:"is_liked,omitempty"`
	MyState     *MyCommentState  `json:"my_state,omitempty"`
	// TopReplies 仅在 comments/list 返回的顶层评论中填充(前 N 条回复预览)。
	TopReplies []CommentItem `json:"top_replies,omitempty"`
}

type CreateCommentRequest struct {
	MomentID string   `json:"moment_id"`
	Content  string   `json:"content"`
	ParentID string   `json:"parent_id,omitempty"`
	Mentions []string `json:"mentions,omitempty"`
}

type ListCommentsRequest struct {
	MomentID   string            `json:"moment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PaginatedComments struct {
	Items      []CommentItem `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type DeleteCommentRequest struct {
	CommentID string `json:"comment_id"`
}

type PaginationParams = types.PaginationParams

type CommentRepliesRequest struct {
	CommentID  string            `json:"comment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type LikeCommentRequest struct {
	CommentID string `json:"comment_id"`
}

type UnlikeCommentRequest struct {
	CommentID string `json:"comment_id"`
}

type CommentLikesRequest struct {
	CommentID  string            `json:"comment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type CommentLikeItem struct {
	User    UserInfo `json:"user"`
	LikedAt string   `json:"liked_at"`
}

type PaginatedCommentLikes struct {
	Items      []CommentLikeItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}
