package reactions

import "hotspot/backend-go/internal/platform/types"

type UserInfo = types.UserInfo
type PaginationParams = types.PaginationParams

// ReactTarget enumerates the resource kinds that accept emoji reactions.
// Matches spec api_spec_types.md ReactTarget.
type ReactTarget string

const (
	ReactTargetMessage ReactTarget = "message"
	ReactTargetMoment  ReactTarget = "moment"
	ReactTargetComment ReactTarget = "comment"
)

// ValidReactTarget reports whether t is one of the supported reaction targets.
func ValidReactTarget(t ReactTarget) bool {
	switch t {
	case ReactTargetMessage, ReactTargetMoment, ReactTargetComment:
		return true
	}
	return false
}

// Reaction is the per-emoji aggregate attached to MomentItem.reactions /
// CommentItem.reactions (spec api_spec_types.md Reaction). ReactedByMe is
// @auth-only semantics: false for anonymous viewers.
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

// ReactionItem is a single row in reactions/fetch (spec api_spec_types.md
// ReactionItem).
type ReactionItem struct {
	User      UserInfo `json:"user"`
	Emoji     string   `json:"emoji"`
	ReactedAt string   `json:"reacted_at"`
}

// SetReactionRequest backs POST /api/v1/reactions/set (spec api_spec_reaction.md).
// Active=true adds/switches the caller's emoji for the target; active=false
// removes it.
type SetReactionRequest struct {
	TargetType ReactTarget `json:"target_type"`
	TargetID   string      `json:"target_id"`
	Emoji      string      `json:"emoji"`
	Active     bool        `json:"active"`
}

// ListReactionsRequest backs POST /api/v1/reactions/fetch.
type ListReactionsRequest struct {
	TargetType ReactTarget       `json:"target_type"`
	TargetID   string            `json:"target_id"`
	Emoji      string            `json:"emoji,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// PaginatedReactions wraps a reactions/fetch page.
type PaginatedReactions struct {
	Items      []ReactionItem `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}
