package comments

import "hotspot/backend-go/internal/platform/types"

type UserInfo = types.UserInfo

// MyCommentState removed — likes_count/my_state 不在契约 CommentItem，仅 repo 内部使用。

// Reaction mirrors spec api_spec_types.md Reaction (per-emoji aggregate with
// reacted_by_me). Defined here rather than imported from the reactions module
// so comments does not depend on reactions at the type level; the values are
// filled by a reactions.Service adapter at read time.
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

// CommentItem 评论统一公开类型（spec api_spec_types.md CommentItem）。
// 恒定 nullable：parent_id 恒在（nil = 顶层评论）；mentions/reactions/top_replies 恒为数组。
// likes_count/my_state 不在契约类型，仅供内部/兼容，不再对外序列化。
type CommentItem struct {
	CommentID string   `json:"comment_id"`
	User      UserInfo `json:"user"`
	Content   string   `json:"content"`
	// parent_id 恒定 nullable：nil = 顶层评论。
	ParentID  *string  `json:"parent_id"`
	Mentions  []string `json:"mentions"`
	CreatedAt string   `json:"created_at"`
	// RepliesCount 子评论数量。
	RepliesCount int  `json:"replies_count"`
	IsDeleted    bool `json:"is_deleted"`
	// reactions 恒为数组（空数组 = 无反应）。
	Reactions []Reaction `json:"reactions"`
	// top_replies 恒为数组；仅 comments/fetch 返回顶层评论的回复预览，其它端点为空数组。
	TopReplies []CommentItem `json:"top_replies"`
	// MomentID 仅内部使用（评论挂载瞬间），不对外序列化。
	MomentID string `json:"-"`
	// LikesCount 兼容保留（reactions 聚合之前的历史计数），不对外序列化。
	LikesCount int `json:"-"`
}

// strPtr 构造 string 指针（用于 CommentItem.ParentID 恒定 nullable）。
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// parentIDStr 将 *string 归一为 any（nil→nil），供测试行构造直接传给 pgx mock。
func parentIDStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
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
	NextCursor *string       `json:"next_cursor"`
}

type DeleteCommentRequest struct {
	CommentID string `json:"comment_id"`
}

// GetCommentRequest backs POST /api/v1/comments/open_the_door
// (spec api_spec_moment.md CommentOpenTheDoorRequest).
type GetCommentRequest struct {
	CommentID string `json:"comment_id"`
}

type PaginationParams = types.PaginationParams

type CommentRepliesRequest struct {
	CommentID  string            `json:"comment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}
