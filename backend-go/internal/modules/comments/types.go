package comments

import "hotspot/backend-go/internal/platform/types"

type UserInfo = types.UserInfo

// MyCommentState holds the requesting user's per-comment interaction state.
// The struct is intentionally kept (and may be empty) so callers that allocate
// it do not need to change shape when fields are added back by the reactions
// adapter; IsLiked has been removed in favour of the reactions aggregate.
type MyCommentState struct{}

// Reaction mirrors spec api_spec_types.md Reaction (per-emoji aggregate with
// reacted_by_me). Defined here rather than imported from the reactions module
// so comments does not depend on reactions at the type level; the values are
// filled by a reactions.Service adapter at read time.
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

type CommentItem struct {
	CommentID    string   `json:"comment_id"`
	MomentID     string   `json:"-"`
	User         UserInfo `json:"user"`
	Content      string   `json:"content"`
	ParentID     string   `json:"parent_id,omitempty"`
	Mentions     []string `json:"mentions,omitempty"`
	CreatedAt    string   `json:"created_at"`
	LikesCount   int      `json:"likes_count"`
	RepliesCount int      `json:"replies_count"`
	IsDeleted    bool     `json:"is_deleted"`
	// Reactions is the per-emoji aggregate (@auth-only reacted_by_me). Omitted
	// when the comment has no reactions. LikesCount above is the total across
	// all emojis, kept for legacy clients.
	Reactions []Reaction      `json:"reactions,omitempty"`
	MyState   *MyCommentState `json:"my_state,omitempty"`
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

// GetCommentRequest backs POST /api/v1/comments/get_detail
// (spec api_spec_moment.md CommentOpenTheDoorRequest).
type GetCommentRequest struct {
	CommentID string `json:"comment_id"`
}

type PaginationParams = types.PaginationParams

type CommentRepliesRequest struct {
	CommentID  string            `json:"comment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}
