package comments

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

// ── 匿名节点（/public/v1）专属类型 ──────────────────────────────────────
// spec api_spec_public.md「类型体系」：匿名版 Request/Response 使用 Public* 类型，
// 响应不含查看者状态字段；reactions 换为 PublicReaction（去 reacted_by_me）。

type PublicReaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

type PublicCommentItem struct {
	CommentID    string              `json:"comment_id"`
	User         UserInfo            `json:"user"`
	Content      string              `json:"content"`
	ParentID     *string             `json:"parent_id"`
	Mentions     []string            `json:"mentions"`
	CreatedAt    string              `json:"created_at"`
	RepliesCount int                 `json:"replies_count"`
	Reactions    []PublicReaction    `json:"reactions"`
	IsDeleted    bool                `json:"is_deleted"`
	TopReplies   []PublicCommentItem `json:"top_replies"`
}

type PublicPaginatedComments struct {
	Items      []PublicCommentItem `json:"items"`
	NextCursor *string             `json:"next_cursor"`
}

type PublicFetchCommentsRequest struct {
	MomentID   string            `json:"moment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchCommentRepliesRequest struct {
	CommentID  string            `json:"comment_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicOpenCommentDoorRequest struct {
	CommentID string `json:"comment_id"`
}

func toPublicCommentItem(item CommentItem) PublicCommentItem {
	mentions := item.Mentions
	if mentions == nil {
		mentions = []string{}
	}
	reactions := make([]PublicReaction, 0, len(item.Reactions))
	for _, reaction := range item.Reactions {
		reactions = append(reactions, PublicReaction{Emoji: reaction.Emoji, Count: reaction.Count})
	}
	topReplies := make([]PublicCommentItem, 0, len(item.TopReplies))
	for _, reply := range item.TopReplies {
		topReplies = append(topReplies, toPublicCommentItem(reply))
	}
	return PublicCommentItem{
		CommentID:    item.CommentID,
		User:         item.User,
		Content:      item.Content,
		ParentID:     item.ParentID,
		Mentions:     mentions,
		CreatedAt:    item.CreatedAt,
		RepliesCount: item.RepliesCount,
		Reactions:    reactions,
		IsDeleted:    item.IsDeleted,
		TopReplies:   topReplies,
	}
}

func toPublicPaginatedComments(page PaginatedComments) PublicPaginatedComments {
	items := make([]PublicCommentItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toPublicCommentItem(item))
	}
	return PublicPaginatedComments{Items: items, NextCursor: page.NextCursor}
}

// ── 匿名端点 handler（仅挂在 /public/v1/comments/*）────────────────────

func (h *Handler) PublicFetch(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchCommentsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || pubReq.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "PublicFetch", "参数错误")
		return
	}
	list, err := h.service.List(r.Context(), ListCommentsRequest{MomentID: pubReq.MomentID, Pagination: pubReq.Pagination})
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			httpx.WriteBusinessError(w, r, "PublicFetch", httpx.CodeNotFound, "瞬间不存在")
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteInvalidRequest(w, r, "PublicFetch", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "PublicFetch", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicPaginatedComments(list)))
}

func (h *Handler) PublicFetchReplies(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchCommentRepliesRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || pubReq.CommentID == "" {
		httpx.WriteInvalidRequest(w, r, "PublicFetchReplies", "参数错误")
		return
	}
	list, err := h.service.ListReplies(r.Context(), CommentRepliesRequest{CommentID: pubReq.CommentID, Pagination: pubReq.Pagination})
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteBusinessError(w, r, "PublicFetchReplies", httpx.CodeNotFound, "评论不存在")
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteInvalidRequest(w, r, "PublicFetchReplies", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "PublicFetchReplies", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicPaginatedComments(list)))
}

func (h *Handler) PublicOpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicOpenCommentDoorRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || pubReq.CommentID == "" {
		httpx.WriteInvalidRequest(w, r, "PublicOpenTheDoor", "参数错误")
		return
	}
	item, found, err := h.service.GetForDetail(r.Context(), pubReq.CommentID)
	if err != nil {
		httpx.WriteAPIError(w, r, "PublicOpenTheDoor", err)
		return
	}
	if !found {
		httpx.WriteBusinessError(w, r, "PublicOpenTheDoor", httpx.CodeNotFound, "评论不存在")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicCommentItem(item)))
}
